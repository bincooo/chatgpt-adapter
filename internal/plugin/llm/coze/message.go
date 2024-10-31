package coze

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"errors"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func calcTokens(messages []coze.Message) (tokensL int) {
	for _, message := range messages {
		tokensL += common.CalcTokens(message.Content)
	}
	return
}

func waitMessage(chatResponse chan string, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if strings.HasPrefix(message, "error: ") {
			return "", logger.WarpError(
				errors.New(strings.TrimPrefix(message, "error: ")),
			)
		}

		message = strings.TrimPrefix(message, "text: ")
		if len(message) > 0 {
			content += message
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, cancel chan error, chatResponse chan string, sse bool) (content string) {
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)

	for {
		select {
		case err := <-cancel:
			if err != nil {
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					logger.Error(err)
					response.Error(ctx, -1, err)
				}
				return
			}
			goto label
		default:
			raw, ok := <-chatResponse
			if !ok {
				raw = common.ExecMatchers(matchers, "", true)
				if raw != "" && sse {
					response.SSEResponse(ctx, Model, raw, created)
				}
				content += raw
				goto label
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

			raw = common.ExecMatchers(matchers, raw, false)
			if len(raw) == 0 {
				continue
			}

			if sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
		}
	}

label:
	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

func mergeMessages(ctx *gin.Context) (newMessages []coze.Message, tokens int, err error) {
	var (
		completion = common.GetGinCompletion(ctx)
		messages   = completion.Messages
	)

	var (
		pos      = 0
		contents []string
	)
	messageL := len(messages)
	for {
		if pos > messageL-1 {
			break
		}

		message := messages[pos]
		role, end := common.ConvertRole(ctx, message.GetString("role"))
		contents = append(contents, role+message.GetString("content")+end)
		pos++
	}

	message := strings.Join(contents, "")
	if strings.HasSuffix(message, "<|end|>\n\n") {
		message = message[:len(message)-9]
	}

	tokens += common.CalcTokens(message)
	newMessages = append(newMessages, coze.Message{
		Role:    "user",
		Content: message,
	})
	return
}
