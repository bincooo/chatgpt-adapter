package bing

import (
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/gin/model"
	"encoding/json"
	"fmt"
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

		if !msg.Is("event", "appendText") {
			continue
		}

		raw := msg.GetString("text")
		logger.Debug("----- raw -----")
		logger.Debug(raw)

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

func echoMessages(ctx *gin.Context, completion model.Completion) {
	content := ""
	var (
		toolMessages = toolcall.ExtractToolMessages(&completion)
	)

	if response.IsClaude(ctx, completion.Model) {
		content = completion.Messages[0].GetString("content")
	} else {
		chunkBytes, _ := json.MarshalIndent(completion.Messages, "", "  ")
		content += string(chunkBytes)
	}

	if len(toolMessages) > 0 {
		content += "\n----------toolCallMessages----------\n"
		chunkBytes, _ := json.MarshalIndent(toolMessages, "", "  ")
		content += string(chunkBytes)
	}

	response.Echo(ctx, completion.Model, content, completion.Stream)
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
