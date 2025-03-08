package bing

import (
	"chatgpt-adapter/core/gin/model"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/iocgo/sdk/env"
	"net/http"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
)

const (
	ginTokens = "__tokens__"
)

func waitMessage(message chan []byte, cancel func(str string) bool) (content string, err error) {
	for {
		chunk, ok := <-message
		if !ok {
			break
		}

		magic := chunk[0]
		chunk = chunk[1:]
		if magic == 1 {
			err = fmt.Errorf("%s", chunk)
			break
		}

		var msg model.Keyv[interface{}]
		err = json.Unmarshal(chunk, &msg)
		if err != nil {
			logger.Error(err)
			continue
		}

		if !msg.Is("event", "appendText") {
			continue
		}

		raw := msg.GetString("text")
		logger.Debug("----- raw -----")
		logger.Debug(raw)
		content += raw
		if cancel != nil && cancel(content) {
			return content, nil
		}
	}
	return
}

func waitResponse(ctx *gin.Context, message chan []byte, sse bool) (content string) {
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)
	onceExec := sync.OnceFunc(func() {
		if !sse {
			ctx.Writer.WriteHeader(http.StatusOK)
		}
	})

	var (
		matchers = common.GetGinMatchers(ctx)
	)

	for {

		chunk, ok := <-message
		if !ok {
			raw := response.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			break
		}

		magic := chunk[0]
		chunk = chunk[1:]
		if magic == 1 {
			asError(ctx, string(chunk))
			break
		}

		var msg model.Keyv[interface{}]
		err := json.Unmarshal(chunk, &msg)
		if err != nil {
			logger.Error(err)
			continue
		}

		raw := ""
		if msg.Is("event", "appendText") {
			raw = msg.GetString("text")
		}
		if msg.Is("event", "imageGenerated") {
			raw = fmt.Sprintf("![image](%s)", msg.GetString("url"))
		}
		if msg.Is("event", "replaceText") {
			raw = msg.GetString("text")
		}
		if len(raw) == 0 {
			continue
		}

		logger.Debug("----- raw -----")
		logger.Debug(raw)
		onceExec()

		raw = response.ExecMatchers(matchers, raw, false)
		if len(raw) == 0 {
			continue
		}

		if raw == response.EOF {
			break
		}

		if sse {
			response.SSEResponse(ctx, Model, raw, created)
		}
		content += raw
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}
	ctx.Set(vars.GinCompletionUsage, response.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

func asError(ctx *gin.Context, msg interface{}) {
	if msg == nil || msg == "" {
		return
	}
	logger.Error(msg)
	if response.NotSSEHeader(ctx) {
		response.Error(ctx, -1, msg)
	}
	return
}

func hookCloudflare() (challenge string, err error) {
	baseUrl := env.Env.GetString("browser-less.reversal")
	if !env.Env.GetBool("browser-less.enabled") && baseUrl == "" {
		return "", errors.New("trying cloudflare failed, please setting `browser-less.enabled` or `browser-less.reversal`")
	}

	logger.Info("trying cloudflare ...")
	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + env.Env.GetString("browser-less.port")
	}

	r, err := emit.ClientBuilder(common.HTTPClient).
		GET(baseUrl+"/v0/turnstile").
		Header("sitekey", "0x4AAAAAAAg146IpY3lPNWte").
		Header("website", "https://copilot.microsoft.com").
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		logger.Error(err)
		if emit.IsJSON(r) == nil {
			logger.Error(emit.TextResponse(r))
		}
		return
	}

	defer r.Body.Close()
	obj, err := emit.ToMap(r)
	if err != nil {
		logger.Error(err)
		return
	}

	if data, ok := obj["data"].(string); ok {
		challenge = data
		return
	}

	msg := "challenge failed"
	if data, ok := obj["msg"].(string); ok {
		msg = data
	}
	err = errors.New(msg)
	return
}
