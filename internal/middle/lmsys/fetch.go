package lmsys

import (
	"context"
	"errors"
	com "github.com/bincooo/chatgpt-adapter/v2/internal/common"
	emits "github.com/bincooo/gio.emits"
	"github.com/bincooo/gio.emits/common"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	baseUrl = "https://arena.lmsys.org"
	ua      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
)

var cookies string

type options struct {
	model       string
	temperature float32
	topP        float32
	maxTokens   int

	cancel chan bool
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
	co, err := partOne(ctx, proxies, opts.model, messages, hash)
	if err != nil {
		return nil, err
	}

	if co == "" {
		return nil, errors.New("fetch failed")
	}

	cookies = co
	return partTwo(ctx, proxies, hash, opts)
}

func partTwo(ctx context.Context, proxies, hash string, opts options) (chan string, error) {
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
		Header("Origin", "https://arena.lmsys.org").
		Header("Referer", "https://arena.lmsys.org/").
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

	cookies = common.MergeCookies(cookies, common.GetCookies(response))
	response, err = common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		Header("Origin", "https://arena.lmsys.org").
		Header("Referer", "https://arena.lmsys.org/").
		DoWith(http.StatusOK)
	if err != nil {
		return nil, err
	}

	cookies = common.MergeCookies(cookies, common.GetCookies(response))
	e, err := emits.New(ctx, response)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	pos := 0

	e.Event("process_generating", func(j emits.JoinCompleted) interface{} {
		select {
		case <-opts.cancel:
			e.Cancel()
			return nil
		default:
			data := j.Output.Data
			if len(data) < 2 {
				e.Error(errors.New("illegal response"))
				return nil
			}

			items, ok := data[1].([]interface{})
			if !ok {
				e.Error(errors.New("illegal response"))
				return nil
			}

			if len(items) < 1 {
				e.Error(errors.New("illegal response"))
				return nil
			}

			items, ok = items[0].([]interface{})
			if !ok {
				e.Error(errors.New("illegal response"))
				return nil
			}

			if len(items) < 3 {
				return nil
			}

			if items[0] != "replace" {
				e.Error(errors.New("illegal response"))
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
		}
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

	if cookies == "" {
		cookies = "SERVERID=S2|" + com.RandStr(5)
	}

	response, err := common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		POST(baseUrl+"/queue/join").
		JHeader().
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		Header("Origin", "https://arena.lmsys.org").
		Header("Referer", "https://arena.lmsys.org/").
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

	cookies = common.MergeCookies(cookies, common.GetCookies(response))
	response, err = common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		Header("Origin", "https://arena.lmsys.org").
		Header("Referer", "https://arena.lmsys.org/").
		DoWith(http.StatusOK)
	if err != nil {
		return "", err
	}

	cookies = common.MergeCookies(cookies, common.GetCookies(response))
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
