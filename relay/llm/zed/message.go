package zed

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
)

const (
	ginTokens = "__tokens__"
)

type zedResponse struct {
	Type         string      `json:"type"`
	Index        int         `json:"index"`
	ContentBlock interface{} `json:"content_block"`
	Delta        struct {
		Type         string `json:"type"`
		Text         string `json:"text"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
	} `json:"delta"`
	Message struct {
		Id      string        `json:"id"`
		Type    string        `json:"type"`
		Role    string        `json:"role"`
		Content []interface{} `json:"content"`
		Model   string        `json:"model"`
		Usage   struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
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

		var res zedResponse
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		delta := res.Delta
		if delta.StopReason == "end_turn" {
			break
		}

		if delta.Type != "text_delta" {
			continue
		}

		raw := delta.Text
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

		var res zedResponse
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		delta := res.Delta
		if delta.StopReason == "end_turn" {
			break
		}

		if delta.Type != "text_delta" {
			continue
		}

		raw := delta.Text
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
