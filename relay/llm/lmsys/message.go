package lmsys

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"sync"
	"time"
)

const ginTokens = "__tokens__"

func waitMessage(chatResponse chan string, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if strings.HasPrefix(message, "error: ") {
			return "", errors.New(strings.TrimPrefix(message, "error: "))
		}

		message = strings.TrimPrefix(message, "text: ")
		logger.Debug("----- raw -----")
		logger.Debug(message)
		if len(message) > 0 {
			content += message
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, chatResponse chan string, sse bool) (content string) {
	created := time.Now().Unix()
	logger.Info("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)
	matchers := common.GetGinMatchers(ctx)
	onceExec := sync.OnceFunc(func() {
		if !sse {
			ctx.Writer.WriteHeader(http.StatusOK)
		}
	})

	for {
		raw, ok := <-chatResponse
		if !ok {
			raw = response.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			break
		}

		if strings.HasPrefix(raw, "error: ") {
			err := strings.TrimPrefix(raw, "error: ")
			logger.Error(err)
			if response.NotSSEHeader(ctx) {
				logger.Error(err)
				response.Error(ctx, -1, err)
			}
			return
		}

		raw = strings.TrimPrefix(raw, "text: ")
		contentL := len(raw)
		if contentL <= 0 {
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

		if sse && len(raw) > 0 {
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

func mergeMessages(ctx *gin.Context, completion model.Completion) (newMessages string, err error) {
	var (
		messages    = completion.Messages
		specialized = ctx.GetBool("specialized")
		isC         = response.IsClaude(ctx, completion.Model)
	)

	messageL := len(messages)
	if specialized && isC && messageL == 1 {
		newMessages = messages[0].GetString("content")
		return
	}

	var (
		pos      = 0
		contents []string
	)

	for {
		if pos > messageL-1 {
			break
		}

		message := messages[pos]
		role, end := response.ConvertRole(ctx, message.GetString("role"))
		contents = append(contents, role+message.GetString("content")+end)
		pos++
	}

	newMessages = strings.Join(contents, "")
	if strings.HasSuffix(newMessages, "<|end|>\n\n") {
		newMessages = newMessages[:len(newMessages)-9]
	}
	return
}
