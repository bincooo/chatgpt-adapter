package lmsys

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
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
		if len(message) > 0 {
			content += message
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan string, cancel chan error, sse bool) {
	content := ""
	created := time.Now().Unix()
	logger.Info("waitResponse ...")
	tokens := ctx.GetInt("tokens")

	for {
		select {
		case err := <-cancel:
			if err != nil {
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, err)
				}
				logger.Error(err)
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
			if sse && len(raw) > 0 {
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

func mergeMessages(messages []pkg.Keyv[interface{}]) (newMessages string) {
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "tool", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	slices := common.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []string {
		role := message["role"]
		if condition(role) == condition(next) {
			// cache buffer
			if role == "function" || role == "tool" {
				buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], message["content"]))
				return nil
			}

			buffer.WriteString(fmt.Sprintf("<|%s|>\n%s\n<|end|>", role, message["content"]))
			return nil
		}

		defer buffer.Reset()
		var result []string
		if previous == "system" {
			result = append(result, fmt.Sprintf("<|system|>\n%s\n<|end|>", buffer.String()))
			result = append(result, "<|assistant|>ok ~<|end|>\n")
			buffer.Reset()
		}

		buffer.WriteString(fmt.Sprintf("<|%s|>\n%s\n<|end|>", role, message["content"]))
		return append(result, buffer.String())
	})

	newMessages = strings.Join(slices, "\n\n")
	newMessages += "\n<|assistant|>"
	return
}
