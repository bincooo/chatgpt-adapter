package deepseek

import (
	"bufio"
	"bytes"
	"chatgpt-adapter/core/gin/model"
	"encoding/json"
	"github.com/iocgo/sdk/env"
	"io"
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

func waitMessage(r *http.Response, cancel func(str string) bool) (content string, err error) {
	defer r.Body.Close()
	reader := bufio.NewReader(r.Body)
	var dataBytes []byte
	for {
		dataBytes, _, err = reader.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return
		}

		var res model.Response
		if bytes.HasPrefix(dataBytes, []byte("data: ")) {
			dataBytes = dataBytes[6:]
		}
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		if len(res.Choices) == 0 {
			continue
		}

		if res.Choices[0].FinishReason != nil && *res.Choices[0].FinishReason == "stop" {
			break
		}

		delta := res.Choices[0].Delta
		if delta.Type == "thinking" {
			continue
		}

		raw := delta.Content
		logger.Debug("----- raw -----")
		logger.Debug(raw)
		content += raw
		if cancel != nil && cancel(content) {
			return content, nil
		}
	}
	return
}

func waitResponse(ctx *gin.Context, r *http.Response, sse bool) (content string) {
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)
	thinkReason := env.Env.GetBool("server.think_reason")
	reasoningContent := ""

	onceExec := sync.OnceFunc(func() {
		if !sse {
			ctx.Writer.WriteHeader(http.StatusOK)
		}
	})

	var (
		matchers = common.GetGinMatchers(ctx)
	)

	defer r.Body.Close()
	reader := bufio.NewReader(r.Body)
	think := 0
	for {
		dataBytes, _, err := reader.ReadLine()
		if err == io.EOF {
			raw := response.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			break
		}

		if asError(ctx, err) {
			return
		}

		var res model.Response
		if bytes.HasPrefix(dataBytes, []byte("data: ")) {
			dataBytes = dataBytes[6:]
		}
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		if len(res.Choices) == 0 {
			continue
		}

		if res.Choices[0].FinishReason != nil && *res.Choices[0].FinishReason == "stop" {
			break
		}

		delta := res.Choices[0].Delta
		if delta.Type == "thinking" {
			if thinkReason {
				delta.ReasoningContent = delta.Content
				reasoningContent += delta.Content
				delta.Content = ""
				think = 1
			} else if think == 0 {
				think = 1
				delta.Content = "<think>\n" + delta.Content
			}
		} else {
			if thinkReason {
				think = 2
			} else if think == 1 {
				think = 2
				delta.Content = "\n</think>\n" + delta.Content
			}
		}

		raw := delta.Content
		if thinkReason && think == 1 {
			logger.Debug("----- think raw -----")
			logger.Debug(delta.ReasoningContent)
			goto label
		}

		logger.Debug("----- raw -----")
		logger.Debug(raw)
		onceExec()

		raw = response.ExecMatchers(matchers, raw, false)
		if len(raw) == 0 {
			continue
		}

	label:
		if raw == response.EOF {
			break
		}

		if sse {
			response.ReasonSSEResponse(ctx, Model, raw, delta.ReasoningContent, created)
		}
		content += raw
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}
	ctx.Set(vars.GinCompletionUsage, response.CalcUsageTokens(reasoningContent+content, tokens))
	if !sse {
		response.ReasonResponse(ctx, Model, content, reasoningContent)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

func asError(ctx *gin.Context, err error) (ok bool) {
	if err == nil {
		return
	}

	logger.Error(err)
	if response.NotSSEHeader(ctx) {
		response.Error(ctx, -1, err)
	}
	ok = true
	return
}
