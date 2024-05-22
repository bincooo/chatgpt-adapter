package coze

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
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
			return "", errors.New(strings.TrimPrefix(message, "error: "))
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

func waitResponse(ctx *gin.Context, matchers []common.Matcher, cancel chan error, chatResponse chan string, sse bool) {
	content := ""
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)

	for {
		select {
		case err := <-cancel:
			if err != nil {
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, err)
				}
				return
			}
			goto label
		default:
			raw, ok := <-chatResponse
			if !ok {
				goto label
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

			raw = common.ExecMatchers(matchers, raw)
			if sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
		}
	}

label:
	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
}

func mergeMessages(messages []pkg.Keyv[interface{}]) (newMessages []coze.Message, tokens int) {
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "assistant", "function", "tool":
			return expr
		default:
			return ""
		}
	}

	newMessages = common.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []coze.Message {
		role := message["role"]
		tokens += common.CalcTokens(message["content"])
		if condition(role) == condition(next) {
			// cache buffer
			if role == "function" || role == "tool" {
				buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], message["content"]))
				return nil
			}
			buffer.WriteString(message["content"])
			return nil
		}

		defer buffer.Reset()
		buffer.WriteString(fmt.Sprintf(message["content"]))
		return []coze.Message{
			{
				Role:    role,
				Content: buffer.String(),
			},
		}
	})

	return
}
