package lmsys

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
			if cancel != nil && !cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []pkg.Matcher, chatResponse chan string, cancel chan error, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")
	tokens := ctx.GetInt("tokens")

	for {
		select {
		case err := <-cancel:
			if err != nil {
				middle.ErrResponse(ctx, -1, err)
				return
			}
			goto label
		default:
			raw, ok := <-chatResponse
			if !ok {
				goto label
			}

			if strings.HasPrefix(raw, "error: ") {
				logrus.Error(strings.TrimPrefix(raw, "error: "))
				return
			}

			raw = strings.TrimPrefix(raw, "text: ")
			contentL := len(raw)
			if contentL <= 0 {
				continue
			}

			fmt.Printf("----- raw -----\n %s\n", raw)
			raw = pkg.ExecMatchers(matchers, raw)
			if sse && len(raw) > 0 {
				middle.SSEResponse(ctx, Model, raw, nil, created)
			}
			content += raw
		}
	}

label:
	if !sse {
		middle.Response(ctx, Model, content)
	} else {
		middle.SSEResponse(ctx, Model, "[DONE]", common.CalcUsageTokens(content, tokens), created)
	}
}

func mergeMessages(messages []pkg.Keyv[interface{}]) (newMessages string) {
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	slices := common.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []string {
		role := message["role"]
		if condition(role) == condition(next) {
			// cache buffer
			if role == "function" {
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
