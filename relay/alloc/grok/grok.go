package bing

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/proxy"
)

var (
	cookiesContainer *common.PollContainer[string]
	userAgent        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1.1 Safari/605.1.15"
	lang             string
	clearance        string

	mu sync.Mutex
)

func init() {
	inited.AddInitialized(func(env *env.Environment) {
		cookies := env.GetStringSlice("grok.cookies")
		cookiesContainer = common.NewPollContainer[string]("grok", cookies, time.Hour)
		cookiesContainer.Condition = condition
	})
}

func InvocationHandler(ctx *proxy.Context) {
	var (
		gtx  = ctx.In[0].(*gin.Context)
		echo = gtx.GetBool(vars.GinEcho)
	)

	if echo || ctx.Method != "Completion" && ctx.Method != "ToolChoice" {
		ctx.Do()
		return
	}

	logger.Infof("execute static proxy [relay/llm/grok.api]: func %s(...)", ctx.Method)

	if cookiesContainer.Len() == 0 {
		response.Error(gtx, -1, "empty cookies")
		return
	}

	cookie, err := cookiesContainer.Poll(gtx)
	if err != nil {
		logger.Error(err)
		response.Error(gtx, -1, err)
		return
	}
	defer resetMarked(cookie)
	gtx.Set("token", cookie)

	//
	ctx.Do()

	//
	if ctx.Method == "Completion" {
		err = elseOf[error](ctx.Out[0])
	}
	if ctx.Method == "ToolChoice" {
		err = elseOf[error](ctx.Out[1])
	}

	if err != nil {
		logger.Error(err)
		return
	}
}

func condition(cookie string, argv ...interface{}) (ok bool) {
	marker, err := cookiesContainer.Marked(cookie)
	if err != nil {
		logger.Error(err)
		return false
	}

	ok = marker == 0
	if !ok {
		return
	}

	ctx := argv[0].(*gin.Context)
	completion := common.GetGinCompletion(ctx)
	r, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		POST("https://grok.com/rest/rate-limits").
		//Header("accept-language", "en-US,en;q=0.9").
		Header("origin", "https://grok.com").
		Header("referer", "https://grok.com/").
		Header("baggage", "sentry-environment=production,sentry-release="+common.Hex(21)+",sentry-public_key="+strings.ReplaceAll(uuid.NewString(), "-", "")+",sentry-trace_id="+strings.ReplaceAll(uuid.NewString(), "-", "")+",sentry-replay_id="+strings.ReplaceAll(uuid.NewString(), "-", "")+",sentry-sample_rate=1,sentry-sampled=true").
		Header("sentry-trace", strings.ReplaceAll(uuid.NewString(), "-", "")+"-"+common.Hex(16)+"-1").
		Header("user-agent", userAgent).
		Header("accept-language", lang).
		Header("cookie", emit.MergeCookies(cookie, clearance)).
		JSONHeader().
		Body(map[string]interface{}{
			"requestKind": "DEFAULT",
			"modelName":   completion.Model,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		var busErr emit.Error
		if errors.As(err, &busErr) && busErr.Code == 403 {
			_ = hookCloudflare(env.Env)
			ctx.Set("clearance", clearance)
			ctx.Set("userAgent", userAgent)
			ctx.Set("lang", lang)
		}
		logger.Error(err)
		return false
	}

	defer r.Body.Close()
	obj, err := emit.ToMap(r)
	if err != nil {
		logger.Error(err)
		return false
	}

	count := obj["remainingQueries"].(float64)
	ok = count > 0
	if !ok {
		_ = cookiesContainer.MarkTo(cookie, 2)
	}
	return
}

func hookCloudflare(env *env.Environment) error {
	baseUrl := env.GetString("browser-less.reversal")
	if !env.GetBool("browser-less.enabled") && baseUrl == "" {
		return errors.New("trying cloudflare failed, please setting `browser-less.enabled` or `browser-less.reversal`")
	}

	logger.Info("trying cloudflare ...")

	mu.Lock()
	defer mu.Unlock()

	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + env.GetString("browser-less.port")
	}

	r, err := emit.ClientBuilder(common.HTTPClient).
		GET(baseUrl+"/v0/clearance").
		Header("x-website", "https://grok.com").
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

//func cleanCloudflare() {
//	mu.Lock()
//	clearance = ""
//	mu.Unlock()
//}

func resetMarked(cookie string) {
	marker, err := cookiesContainer.Marked(cookie)
	if err != nil {
		logger.Error(err)
		return
	}

	if marker != 1 {
		return
	}

	err = cookiesContainer.MarkTo(cookie, 0)
	if err != nil {
		logger.Error(err)
	}
}

func elseOf[T any](obj any) (zero T) {
	if obj == nil {
		return
	}
	return obj.(T)
}
