package grok

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

const (
	ginTokens = "__tokens__"
)

type grokResponse struct {
	Result struct {
		Response struct {
			Token      string `json:"token"`
			IsThinking bool   `json:"isThinking"`
			IsSoftStop bool   `json:"isSoftStop"`
			ResponseId string `json:"responseId"`
			Title      *struct {
				NewTitle string `json:"newTitle"`
			} `json:"title,omitempty"`
			ModelResponse *map[string]interface{} `json:"modelResponse,omitempty"`
		} `json:"response"`
	} `json:"result"`
}

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

		var res grokResponse
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		if res.Result.Response.IsSoftStop {
			break
		}

		delta := res.Result.Response
		if delta.IsThinking {
			continue
		}

		raw := delta.Token
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

		var res grokResponse
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		if res.Result.Response.IsSoftStop {
			break
		}

		delta := res.Result.Response
		reasonContent := ""
		if delta.IsThinking {
			if thinkReason {
				reasonContent = delta.Token
				reasoningContent += delta.Token
				delta.Token = ""
				think = 1
			} else if think == 0 {
				think = 1
				delta.Token = "<think>\n" + delta.Token
			}
		} else {
			if thinkReason {
				think = 2
			} else if think == 1 {
				think = 2
				delta.Token = "\n</think>\n" + delta.Token
			}
		}

		raw := delta.Token
		if thinkReason && think == 1 {
			logger.Debug("----- think raw -----")
			logger.Debug(reasonContent)
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
			response.ReasonSSEResponse(ctx, Model, raw, reasonContent, created)
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
