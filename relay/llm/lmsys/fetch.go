package lmsys

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/iocgo/sdk/env"
)

const (
	baseUrl = "https://legacy.lmarena.ai"
)

var (
	baseCookies = "_gid=GA1.2.68066840.1717017781; _ga_K6D24EE9ED=GS1.1.1717087813.23.1.1717088648.0.0.0; _gat_gtag_UA_156449732_1=1; _ga_R1FN4KJKJH=GS1.1.1717087813.37.1.1717088648.0.0.0; _ga=GA1.1.1320014795.1715641484"
	ver         = ""

	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
	clearance = ""
	lang      = ""

	mu    sync.Mutex
	state int32 = 0 // 0 常态 1 等待中
)

type options struct {
	model       string
	temperature float32
	topP        float32
	maxTokens   int
	fn          []int
}

func fetch(ctx context.Context, env *env.Environment, proxied, messages string, opts options) (chan string, error) {
	if opts.topP == 0 {
		opts.topP = 1
	}
	if opts.temperature == 0 {
		opts.temperature = 0.7
	}
	if opts.maxTokens == 0 {
		opts.maxTokens = 1024
	}

	hash := emit.GioHash()
	cookies, err := partOne(ctx, env, proxied, &opts, messages, hash)
	if err != nil {
		return nil, err
	}

	if cookies == "" {
		return nil, errors.New("fetch failed")
	}

	err = partTwo(ctx, proxied, cookies, hash, opts)
	if err != nil {
		return nil, err
	}

	return partThree(ctx, proxied, cookies, hash, opts)
}

func partTwo(ctx context.Context, proxied, cookies, hash string, opts options) error {
	obj := map[string]interface{}{
		"event_data":   nil,
		"session_hash": hash,
		"data":         make([]interface{}, 0),
	}

	var response *http.Response
	var err error
	obj["fn_index"] = opts.fn[0] + 1
	obj["trigger_id"] = opts.fn[1]
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST(baseUrl+"/queue/join").
		JSONHeader().
		Ja3().
		Header("User-Agent", userAgent).
		Header("Cookie", cookies).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/").
		Header("Accept-Language", lang).
		Header("Cache-Control", "no-cache").
		Header("Priority", "u=1, i").
		Body(obj).
		DoS(http.StatusOK)
	if err != nil {
		ver = ""
		return err
	}

	obj, err = emit.ToMap(response)
	if err != nil {
		return err
	}

	if eventId, ok := obj["event_id"]; ok {
		logger.Infof("lmsys eventId: %s", eventId)
	} else {
		return errors.New("fetch failed")
	}

	cookies = emit.MergeCookies(cookies, emit.GetCookies(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		Ja3().
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", userAgent).
		Header("Cookie", cookies).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/").
		Header("Accept-Language", lang).
		Header("Cache-Control", "no-cache").
		Header("Priority", "u=1, i").
		DoS(http.StatusOK)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	cookies = emit.MergeCookies(cookies, emit.GetCookies(response))
	e, err := emit.NewGio(ctx, response)
	if err != nil {
		return err
	}

	return e.Do()
}

func partThree(ctx context.Context, proxied, cookies, hash string, opts options) (chan string, error) {
	obj := map[string]interface{}{
		"fn_index":     opts.fn[0] + 2,
		"trigger_id":   opts.fn[1],
		"session_hash": hash,
		"data": []interface{}{
			nil,
			opts.temperature,
			opts.topP,
			opts.maxTokens,
		},
	}

	response, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST(baseUrl+"/queue/join").
		JSONHeader().
		Ja3().
		Header("User-Agent", userAgent).
		Header("Cookie", cookies).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/").
		Header("Accept-Language", lang).
		Header("Cache-Control", "no-cache").
		Header("Priority", "u=1, i").
		Body(obj).
		DoS(http.StatusOK)
	if err != nil {
		return nil, err
	}

	obj, err = emit.ToMap(response)
	_ = response.Body.Close()
	if err != nil {
		return nil, err
	}

	if eventId, ok := obj["event_id"]; ok {
		logger.Infof("lmsys eventId: %s", eventId)
	} else {
		return nil, errors.New("fetch failed")
	}

	cookies = emit.MergeCookies(cookies, emit.GetCookies(response))
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		Ja3().
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", userAgent).
		Header("Cookie", cookies).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/").
		Header("Accept-Language", lang).
		Header("Cache-Control", "no-cache").
		Header("Priority", "u=1, i").
		DoS(http.StatusOK)
	if err != nil {
		return nil, err
	}

	e, err := emit.NewGio(ctx, response)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	pos := 0

	e.Event("*", func(j emit.JoinEvent) (_ interface{}) {
		logger.Tracef("--------- ORIGINAL MESSAGE ---------")
		logger.Tracef("%s", j.InitialBytes)
		return
	})

	e.Event("process_generating", func(j emit.JoinEvent) (_ interface{}) {
		data := j.Output.Data
		if len(data) < 2 {
			return
		}

		items, ok := data[1].([]interface{})
		if !ok {
			return
		}

		if len(items) < 1 {
			return
		}

		items, ok = items[0].([]interface{})
		if !ok {
			return
		}

		if l := len(items); l < 3 {
			if l == 2 {
				str := items[1].(string)
				if !strings.HasPrefix(str, "<span class=") {
					ch <- "error: " + items[1].(string)
				}
			}
			return
		}

		if items[0] != "replace" {
			return
		}

		message := items[2].(string)
		l := len(message)
		if l >= 3 && message[l-3:] == "▌" {
			message = message[:l-3]
			l -= 3
		}

		if pos >= l {
			return
		}

		ch <- "text: " + message[pos:]
		pos = l
		return
	})

	go func() {
		defer close(ch)
		defer response.Body.Close()
		if err = e.Do(); err != nil {
			logger.Error(err)
		}
	}()

	return ch, nil
}

func partOne(ctx context.Context, env *env.Environment, proxied string, opts *options, messages string, hash string) (string, error) {
	obj := map[string]interface{}{
		"event_data":   nil,
		"session_hash": hash,
		"data": []interface{}{
			nil,
			opts.model,
			map[string]string{
				"text": messages,
				// TODO - image
			},
			nil,
		},
	}

	fn := extCookies(env, opts.model)
	if fn == nil {
		return "", errors.New("invalid fn_index & trigger_id")
	}
	var response *http.Response
	var err error
	cookies := fetchCookies(ctx, proxied)
	obj["fn_index"] = fn[0]
	obj["trigger_id"] = fn[1]
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST(baseUrl+"/queue/join").
		JSONHeader().
		Ja3().
		Header("User-Agent", userAgent).
		Header("Cookie", cookies).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/").
		Header("Accept-Language", lang).
		Header("Cache-Control", "no-cache").
		Header("Priority", "u=1, i").
		Body(obj).
		DoS(http.StatusOK)
	if err != nil {
		ver = ""
		return "", err
	}

	obj, err = emit.ToMap(response)
	if err != nil {
		return "", err
	}

	if eventId, ok := obj["event_id"]; ok {
		logger.Infof("lmsys eventId: %s", eventId)
	} else {
		return "", errors.New("fetch failed")
	}

	cookies = emit.MergeCookies(cookies, emit.GetCookies(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		Ja3().
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", userAgent).
		Header("Cookie", cookies).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/").
		Header("Accept-Language", lang).
		Header("Cache-Control", "no-cache").
		Header("Priority", "u=1, i").
		DoS(http.StatusOK)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	cookies = emit.MergeCookies(cookies, emit.GetCookies(response))
	e, err := emit.NewGio(ctx, response)
	if err != nil {
		return "", err
	}

	next := false
	e.Event("process_completed", func(j emit.JoinEvent) interface{} {
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

func extCookies(env *env.Environment, model string) (fn []int) {
	token := env.GetString("lmsys.token")
	fn = []int{-1, -1}

	exec := func() (ret bool) {
		var obj interface{}
		var slice []interface{}
		err := json.Unmarshal([]byte(token), &obj)
		if err != nil {
			logger.Error(err)
			return
		}

		dict, ok := obj.(map[string]interface{})
		if ok {
			obj, ok = dict[model]
			if !ok {
				return
			}
		}

		slice, ok = obj.([]interface{})
		if ok {
			if len(slice) < 2 {
				logger.Errorf("%s len < 2", token)
				return
			}
			fn = []int{int(slice[0].(float64)), int(slice[1].(float64))}
			return true
		}
		return
	}

	if len(token) > 2 && ((token[0] == '[' && token[len(token)-1] == ']') || (token[0] == '{' && token[len(token)-1] == '}')) {
		if exec() {
			return
		}
	}

	exec()
	return
}

func fetchCookies(ctx context.Context, proxied string) (cookies string) {
	if ver != "" {
		cookies = fmt.Sprintf("SERVERID=%s|%s", ver, common.Hex(5))
		cookies = emit.MergeCookies(baseCookies, cookies)
		cookies = emit.MergeCookies(cookies, clearance)
		return
	}
	retry := 3
label:
	if retry <= 0 {
		return
	}
	retry--
	response, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		GET(baseUrl+"/info").
		Ja3().
		Header("pragma", "no-cache").
		Header("cache-control", "no-cache").
		Header("Accept-Language", lang).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/").
		Header("priority", "u=1, i").
		Header("cookie", emit.MergeCookies(baseCookies, clearance)).
		Header("User-Agent", userAgent).
		DoS(http.StatusOK)
	if err != nil {
		var emitErr emit.Error
		// 人机验证
		if errors.As(err, &emitErr) && emitErr.Code == 403 {
			err = hookCloudflare(env.Env)
			goto label
		}
		logger.Error(err)
		return
	}

	_ = response.Body.Close()
	cookie := emit.GetCookie(response, "SERVERID")
	if cookie == "" {
		goto label
	}

	co := strings.Split(cookie, "|")
	if len(co) < 2 || len(co[0]) < 1 || co[0][0] != 'S' || co[0] == "S0" {
		goto label
	}

	ver = co[0]
	cookies = fmt.Sprintf("SERVERID=%s|%s", ver, common.Hex(5))
	cookies = emit.MergeCookies(baseCookies, clearance)
	return
}

func hookCloudflare(env *env.Environment) error {
	atomic.CompareAndSwapInt32(&state, 0, 1)

	reversalUrl := env.GetString("browser-less.reversal")
	if !env.GetBool("browser-less.enabled") && reversalUrl == "" {
		return errors.New("trying cloudflare failed, please setting `browser-less.enabled` or `browser-less.reversal`")
	}

	mu.Lock()
	defer mu.Unlock()
	if state != 1 {
		return nil
	}

	defer func() { state = 0 }()

	logger.Info("trying cloudflare ...")

	if reversalUrl == "" {
		reversalUrl = "http://127.0.0.1:" + env.GetString("browser-less.port")
	}

	r, err := emit.ClientBuilder(common.HTTPClient).
		Header("x-website", baseUrl).
		GET(reversalUrl+"/v0/clearance").
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		logger.Error(err)
		if emit.IsJSON(r) == nil {
			logger.Error(emit.TextResponse(r))
		}
		return err
	}

	defer r.Body.Close()
	obj, err := emit.ToMap(r)
	if err != nil {
		logger.Error(err)
		return err
	}

	data := obj["data"].(map[string]interface{})
	clearance = data["cookie"].(string)
	userAgent = data["userAgent"].(string)
	lang = data["lang"].(string)
	return nil
}
