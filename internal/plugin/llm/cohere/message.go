package cohere

import (
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"errors"
	"fmt"
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

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan string, sse bool) (content string) {
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
				logger.Error(err)
				response.Error(ctx, -1, err)
			}
			return
		}

		if strings.HasPrefix(raw, "tool: ") {
			t := strings.TrimPrefix(raw, "tool: ")
			if err := json.Unmarshal([]byte(t), &functionCall); err != nil {
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					logger.Error(err)
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
		if len(raw) == 0 {
			continue
		}

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

func mergeMessages(ctx *gin.Context, messages []pkg.Keyv[interface{}]) (content string) {
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "assistant", "function", "tool":
			return expr
		default:
			return ""
		}
	}

	var (
		user      = ""
		assistant = ""
	)

	{
		keyv, ok := common.GetGinValue[pkg.Keyv[string]](ctx, vars.GinCharSequences)
		if ok {
			user = keyv.GetString("user")
			assistant = keyv.GetString("assistant")
		}

		if user == "" {
			user = "<|user|>"
		}
		if assistant == "" {
			assistant = "<|assistant|>"
		}
	}

	tor := func(r string) string {
		switch r {
		case "user":
			return user
		case "assistant":
			return assistant
		default:
			return "<|" + r + "|>"
		}
	}

	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (messages []map[string]string, _ error) {
		role := opts.Message["role"]
		if condition(role) == condition(opts.Next) {
			// cache buffer
			if role == "function" || role == "tool" {
				opts.Buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", opts.Message["name"], opts.Message["content"]))
				return
			}
			opts.Buffer.WriteString(opts.Message["content"])
			return
		}

		defer opts.Buffer.Reset()
		opts.Buffer.WriteString(fmt.Sprintf(opts.Message["content"]))
		messages = []map[string]string{
			{
				"role":    tor(condition(role)),
				"content": opts.Buffer.String(),
			},
		}
		return
	}

	newMessages, _ := common.TextMessageCombiner(messages, iterator)
	// 尾部添加一个assistant空消息
	if newMessages[len(newMessages)-1]["role"] != "assistant" {
		newMessages = append(newMessages, map[string]string{
			"role":    tor("assistant"),
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
		default:
			return "USER"
		}
	}

	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (result []cohere.Message, _ error) {
		role := opts.Message["role"]
		tokens += common.CalcTokens(opts.Message["content"])
		if opts.Previous == "start" {
			if role == "system" {
				system = opts.Message["content"]
				return
			}
		}

		if condition(role) == condition(opts.Next) {
			// cache buffer
			opts.Buffer.WriteString(opts.Message["content"])
			return
		}

		defer opts.Buffer.Reset()
		opts.Buffer.WriteString(opts.Message["content"])

		if role == "system" {
			result = append(result, cohere.Message{
				Role:    "USER",
				Message: opts.Buffer.String(),
			})
			result = append(result, cohere.Message{
				Role:    "CHATBOT",
				Message: "ok ~",
			})
			return
		}

		if _, ok := opts.Message["toolCalls"]; ok && role == "assistant" {
			return
		}

		if role == "function" || role == "tool" {
			opts.Buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", opts.Message["name"], opts.Message["content"]))
			result = append(result, cohere.Message{
				Role:    condition(role),
				Message: opts.Buffer.String(),
			})
			return
		}

		result = []cohere.Message{
			{
				Role:    condition(role),
				Message: opts.Buffer.String(),
			},
		}
		return
	}

	newMessages, _ = common.TextMessageCombiner(messages, iterator)
	content = "continue"
	if pos := len(newMessages) - 1; pos >= 0 && newMessages[pos].Role == "USER" {
		content = newMessages[pos].Message
		newMessages = newMessages[:pos]
	}
	return
}
