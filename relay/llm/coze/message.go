package coze

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
)

const (
	ginTokens = "__tokens__"
)

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

func mergeMessages(ctx *gin.Context) (newMessages []coze.Message, err error) {
	var (
		completion = common.GetGinCompletion(ctx)
		messages   = completion.Messages
		isC        = response.IsClaude(ctx, completion.Model)
	)

	tokens := 0
	defer func() { ctx.Set(ginTokens, tokens) }()

	messageL := len(messages)
	if isC && messageL == 1 {
		message := messages[0].GetString("content")
		newMessages = append(newMessages, coze.Message{
			Role:    "user",
			Content: message,
		})
		tokens += response.CalcTokens(message)
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
		//if pos == 0 && message.Is("role", "system") {
		//	newMessages = append(newMessages, coze.Message{
		//		Role:    "system",
		//		Content: message.GetString("content"),
		//	})
		//	pos++
		//	continue
		//}

		convertRole, trun := response.ConvertRole(ctx, message.GetString("role"))
		contents = append(contents, convertRole+message.GetString("content")+trun)
		pos++
	}

	message := strings.Join(contents, "")
	tokens += response.CalcTokens(message)
	newMessages = append(newMessages, coze.Message{
		Role:    "user",
		Content: message,
	})
	return
}
