package lmsys

import (
	"context"
	"errors"
	"fmt"
	com "github.com/bincooo/chatgpt-adapter/v2/internal/common"
	emits "github.com/bincooo/gio.emits"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"strings"
)

const (
	baseUrl = "https://arena.lmsys.org"
	ua      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
)

var (
	fns = [][]int{
		{39, 94},
		{41, 95},
	}
)

type options struct {
	model       string
	temperature float32
	topP        float32
	maxTokens   int
	fn          []int
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

	hash := emits.GioHash()
	cookies, err := partOne(ctx, proxies, &opts, messages, hash)
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
		"fn_index":     opts.fn[0] + 1,
		"trigger_id":   opts.fn[1],
		"session_hash": hash,
		"data": []interface{}{
			nil,
			opts.temperature,
			opts.topP,
			opts.maxTokens,
		},
	}

	response, err := emits.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		POST(baseUrl+"/queue/join").
		JHeader().
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		Header("Origin", "https://arena.lmsys.org").
		Header("Referer", "https://arena.lmsys.org/").
		Header("X-Forwarded-For", emits.RandIP()).
		Body(obj).
		DoS(http.StatusOK)
	if err != nil {
		return nil, err
	}

	obj, err = emits.ToMap(response)
	if err != nil {
		return nil, err
	}

	if eventId, ok := obj["event_id"]; ok {
		logrus.Infof("lmsys eventId: %s", eventId)
	} else {
		return nil, errors.New("fetch failed")
	}

	cookies = emits.MergeCookies(cookies, emits.GetCookies(response))
	response, err = emits.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		Header("Origin", "https://arena.lmsys.org").
		Header("Referer", "https://arena.lmsys.org/").
		Header("X-Forwarded-For", emits.RandIP()).
		DoS(http.StatusOK)
	if err != nil {
		return nil, err
	}

	e, err := emits.NewGio(ctx, response)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	pos := 0

	//e.Event("*", func(j emits.JoinCompleted) interface{} {
	//	logrus.Infof("%s", j.InitialBytes)
	//	return nil
	//})
	e.Event("process_generating", func(j emits.JoinEvent) interface{} {
		data := j.Output.Data
		if len(data) < 2 {
			return nil
		}

		items, ok := data[1].([]interface{})
		if !ok {
			return nil
		}

		if len(items) < 1 {
			return nil
		}

		items, ok = items[0].([]interface{})
		if !ok {
			return nil
		}

		if l := len(items); l < 3 {
			if l == 2 {
				str := items[1].(string)
				if !strings.HasPrefix(str, "<span class=") {
					ch <- "error: " + items[1].(string)
				}
			}
			return nil
		}

		if items[0] != "replace" {
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
		if err = e.Do(); err != nil {
			logrus.Error(err)
		}
	}()

	return ch, nil
}

func partOne(ctx context.Context, proxies string, opts *options, messages string, hash string) (string, error) {
	obj := map[string]interface{}{
		"event_data":   nil,
		"session_hash": hash,
		"data": []interface{}{
			nil,
			opts.model,
			messages,
			nil,
		},
	}

	var fn []int
	var response *http.Response
	var err error

	cookies := fmt.Sprintf("SERVERID=S%d|%s", rand.Intn(97)+2, com.RandStr(5))
	for _, fn = range fns {
		obj["fn_index"] = fn[0]
		obj["trigger_id"] = fn[1]

		response, err = emits.ClientBuilder().
			Context(ctx).
			Proxies(proxies).
			POST(baseUrl+"/queue/join?").
			JHeader().
			Header("User-Agent", ua).
			Header("Cookie", cookies).
			Header("Origin", "https://arena.lmsys.org").
			Header("Referer", "https://arena.lmsys.org/").
			Body(obj).
			DoS(http.StatusOK)
		if err == nil {
			break
		}
	}

	if err != nil {
		return "", err
	}

	obj, err = emits.ToMap(response)
	if err != nil {
		return "", err
	}

	if eventId, ok := obj["event_id"]; ok {
		logrus.Infof("lmsys eventId: %s", eventId)
	} else {
		return "", errors.New("fetch failed")
	}

	cookies = emits.MergeCookies(cookies, emits.GetCookies(response))
	response, err = emits.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", ua).
		Header("Cookie", cookies).
		Header("Origin", "https://arena.lmsys.org").
		Header("Referer", "https://arena.lmsys.org/").
		DoS(http.StatusOK)
	if err != nil {
		return "", err
	}

	cookies = emits.MergeCookies(cookies, emits.GetCookies(response))
	e, err := emits.NewGio(ctx, response)
	if err != nil {
		return "", err
	}

	next := false
	e.Event("process_completed", func(j emits.JoinEvent) interface{} {
		next = true
		return nil
	})

	if err = e.Do(); err != nil {
		return "", err
	}

	if !next {
		return "", errors.New("fetch failed")
	}

	opts.fn = fn
	return cookies, nil
}
