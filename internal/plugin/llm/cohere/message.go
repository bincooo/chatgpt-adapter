package cohere

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/cohere-api"
	"github.com/gin-gonic/gin"
	"strings"
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
		if len(message) > 0 {
			content += message
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan string, sse bool) {
	content := ""
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)
	var functionCall []pkg.Keyv[interface{}]

	for {
		raw, ok := <-chatResponse
		if !ok {
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

		if strings.HasPrefix(raw, "tool: ") {
			t := strings.TrimPrefix(raw, "tool: ")
			if err := json.Unmarshal([]byte(t), &functionCall); err != nil {
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, err)
				}
				return
			}
			continue
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

	if functionCall != nil && len(functionCall) > 0 {
		// TODO 一次返回多个暂时这么处理吧
		args, _ := json.Marshal(functionCall[0].GetKeyv("parameters"))
		if sse {
			response.SSEToolCallResponse(ctx, Model, functionCall[0].GetString("name"), string(args), created)
		} else {
			response.ToolCallResponse(ctx, Model, functionCall[0].GetString("name"), string(args))
		}
		return
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
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
		case "end":
			return expr
		case "assistant":
			return "CHATBOT"
		case "system":
			return "SYSTEM"
		case "tool":
			return "TOOL"
		default:
			return "USER"
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
			buffer.WriteString(message["content"])
			return nil
		}

		defer buffer.Reset()
		buffer.WriteString(message["content"])

		var result []cohere.Message
		if role == "system" {
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

		if role == "function" || role == "tool" {
			buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], message["content"]))
			result = append(result, cohere.Message{
				Role:    condition(role),
				Message: buffer.String(),
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
	if pos := len(newMessages) - 1; pos >= 0 && newMessages[pos].Role == "USER" {
		content = newMessages[pos].Message
		newMessages = newMessages[:pos]
	}
	return
}
