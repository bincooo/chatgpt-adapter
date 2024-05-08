package lmsys

import (
	"context"
	"errors"
	emits "github.com/bincooo/gio.emits"
	"github.com/bincooo/gio.emits/common"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"strconv"
)

const (
	baseUrl = "https://arena.lmsys.org"
	ua      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
)

type options struct {
	model       string
	temperature float32
	topP        float32
	maxTokens   int
}

func fetch(ctx context.Context, proxies, messages string, opts options) (chan string, error) {
	if opts.topP == 0 {
		opts.topP = 1
	}
	if opts.temperature == 0 {
		opts.temperature = 0.7
	}
	if opts.maxTokens == 0 {
		opts.maxTokens = 1024
	}

	hash := emits.SessionHash()
	cookies, err := partOne(ctx, proxies, opts.model, messages, hash)
	if err != nil {
		return nil, err
	}

	if cookies == "" {
		return nil, errors.New("fetch failed")
	}

	return partTwo(ctx, proxies, cookies, hash, opts)
}

func partTwo(ctx context.Context, proxies, cookies, hash string, opts options) (chan string, error) {
	obj := map[string]interface{}{
		"event_data":   nil,
		"fn_index":     42,
		"trigger_id":   93,
		"session_hash": hash,
		"data": []interface{}{
			nil,
			opts.temperature,
			opts.topP,
			opts.maxTokens,
		},
	}

	response, err := common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		POST(baseUrl+"/queue/join").
		JHeader().
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		Body(obj).
		DoWith(http.StatusOK)
	if err != nil {
		return nil, err
	}

	obj, err = common.ToMap(response)
	if err != nil {
		return nil, err
	}

	if eventId, ok := obj["event_id"]; ok {
		logrus.Infof("lmsys eventId: %s", eventId)
	} else {
		return nil, errors.New("fetch failed")
	}

	response, err = common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		DoWith(http.StatusOK)
	if err != nil {
		return nil, err
	}

	e, err := emits.New(ctx, response)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	pos := 0

	e.Event("process_generating", func(j emits.JoinCompleted) interface{} {
		data := j.Output.Data
		if len(data) < 2 {
			e.Err = errors.New("illegal response")
			return nil
		}

		items, ok := data[1].([]interface{})
		if !ok {
			e.Err = errors.New("illegal response")
			return nil
		}

		if len(items) < 1 {
			e.Err = errors.New("illegal response")
			return nil
		}

		items, ok = items[0].([]interface{})
		if !ok {
			e.Err = errors.New("illegal response")
			return nil
		}

		if len(items) < 3 {
			return nil
		}

		if items[0] != "replace" {
			e.Err = errors.New("illegal response")
			return nil
		}

		message := items[2].(string)
		l := len(message)
		if message[l-3:] == "▌" {
			message = message[:l-3]
			l -= 3
		}

		if pos >= l {
			return nil
		}

		ch <- "text: " + message[pos:]
		pos = l
		return nil
	})

	go func() {
		defer close(ch)
		_ = e.Do()
	}()

	return ch, nil
}

func partOne(ctx context.Context, proxies string, model string, messages string, hash string) (string, error) {
	obj := map[string]interface{}{
		"event_data":   nil,
		"fn_index":     41,
		"trigger_id":   93,
		"session_hash": hash,
		"data": []interface{}{
			nil,
			model,
			messages,
			nil,
		},
	}

	response, err := common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		POST(baseUrl+"/queue/join").
		JHeader().
		Header("User-Agent", ua).
		Header("Cookie", "SERVERID=S2|"+strconv.Itoa(rand.Intn(899999)+100000)).
		Body(obj).
		DoWith(http.StatusOK)
	if err != nil {
		return "", err
	}

	obj, err = common.ToMap(response)
	if err != nil {
		return "", err
	}

	if eventId, ok := obj["event_id"]; ok {
		logrus.Infof("lmsys eventId: %s", eventId)
	} else {
		return "", errors.New("fetch failed")
	}

	cookies := common.GetCookies(response)
	response, err = common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		DoWith(http.StatusOK)
	if err != nil {
		return "", err
	}

	e, err := emits.New(ctx, response)
	if err != nil {
		return "", err
	}

	next := false
	e.Event("process_completed", func(j emits.JoinCompleted) interface{} {
		next = true
		return nil
	})

	if err = e.Do(); err != nil {
		return "", err
	}

	if !next {
		return "", errors.New("fetch failed")
	}

	return cookies, nil
}
