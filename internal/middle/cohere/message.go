package coh

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/cohere-api"
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
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []pkg.Matcher, chatResponse chan string, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")
	tokens := ctx.GetInt("tokens")

	for {
		raw, ok := <-chatResponse
		if !ok {
			break
		}

		if strings.HasPrefix(raw, "error: ") {
			err := strings.TrimPrefix(raw, "error: ")
			logrus.Error(err)
			if middle.NotSSEHeader(ctx) {
				middle.ErrResponse(ctx, -1, err)
			}
			return
		}

		raw = strings.TrimPrefix(raw, "text: ")
		contentL := len(raw)
		if contentL <= 0 {
			continue
		}

		fmt.Printf("----- raw -----\n %s\n", raw)
		raw = pkg.ExecMatchers(matchers, raw)
		if sse {
			middle.SSEResponse(ctx, Model, raw, created)
		}
		content += raw
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		middle.Response(ctx, Model, content)
	} else {
		middle.SSEResponse(ctx, Model, "[DONE]", created)
	}
}

func mergeMessages(messages []pkg.Keyv[interface{}]) (content string) {
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "assistant", "function", "tool":
			return expr
		default:
			return ""
		}
	}

	newMessages := common.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []map[string]string {
		role := message["role"]
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
		return []map[string]string{
			{
				"role":    condition(role),
				"content": buffer.String(),
			},
		}
	})

	// 尾部添加一个assistant空消息
	if newMessages[len(newMessages)-1]["role"] != "assistant" {
		newMessages = append(newMessages, map[string]string{
			"role":    "assistant",
			"content": "",
		})
	}

	return cohere.MergeMessages(newMessages)
}

func mergeChatMessages(messages []pkg.Keyv[interface{}]) (newMessages []cohere.Message, system, content string, tokens int) {
	condition := func(expr string) string {
		switch expr {
		case "assistant":
			return "Chatbot"
		default:
			return "User"
		}
	}

	index := 0
	newMessages = common.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []cohere.Message {
		role := message["role"]
		tokens += common.CalcTokens(message["content"])
		if index == 0 {
			index++
			if role == "system" {
				system = message["content"]
				return nil
			}
		}

		if condition(role) == condition(next) {
			// cache buffer
			if role == "function" {
				buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], message["content"]))
				return nil
			}
			buffer.WriteString(message["content"])
			return nil
		}

		defer buffer.Reset()
		buffer.WriteString(fmt.Sprintf(message["content"]))

		if role == "system" {
			var result []cohere.Message
			result = append(result, cohere.Message{
				Role:    "User",
				Message: buffer.String(),
			})
			result = append(result, cohere.Message{
				Role:    "Chatbot",
				Message: "ok ~",
			})
			return result
		}

		return []cohere.Message{
			{
				Role:    condition(role),
				Message: buffer.String(),
			},
		}
	})

	content = "continue"
	if idx := len(newMessages) - 1; idx >= 0 && newMessages[idx].Role == "User" {
		content = newMessages[idx].Message
	}
	return
}
