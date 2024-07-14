package you

import (
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func waitMessage(ch chan string, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-ch
		if !ok {
			break
		}

		if strings.HasPrefix(message, "error:") {
			return "", logger.WarpError(errors.New(message[6:]))
		}

		if strings.HasPrefix(message, "limits:") {
			continue
		}

		if len(message) > 0 {
			content += message
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, cancel chan error, ch chan string, sse bool) (content string) {
	var (
		created = time.Now().Unix()
		tokens  = ctx.GetInt(ginTokens)
	)

	logger.Info("waitResponse ...")
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
			message, ok := <-ch
			if !ok {
				goto label
			}

			if strings.HasPrefix(message, "error:") {
				logger.Error(message[6:])
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, message[6:])
				}
				return
			}

			if strings.HasPrefix(message, "limits:") {
				continue
			}

			var raw = message
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

func waitMessageResponse(ctx *gin.Context, ch chan string, matchers []common.Matcher, cancel chan error) (content string) {
	logger.Info("waitResponse ...")
	for {
		select {
		case <-cancel:
			logger.Error("context deadline exceeded")
			if response.NotSSEHeader(ctx) {
				response.Error(ctx, -1, "context deadline exceeded")
			}
			return
		default:
			message, ok := <-ch
			if !ok {
				goto label
			}

			if strings.HasPrefix(message, "error:") {
				logger.Error(message[6:])
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, message[6:])
				}
				return
			}

			if strings.HasPrefix(message, "limits:") {
				continue
			}

			logger.Debug("----- raw -----")
			logger.Debug(message)
			raw := common.ExecMatchers(matchers, message)
			if len(raw) == 0 {
				continue
			}

			response.Event(ctx, "content_block_delta", map[string]interface{}{
				"index": 0,
				"type":  "content_block_delta",
				"delta": map[string]interface{}{
					"type": "text_delta", "text": raw,
				},
			})
			content += raw
		}
	}

label:
	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	response.Event(ctx, "message_delta", map[string]interface{}{
		"type":  "message_delta",
		"usage": map[string]int{"output_tokens": common.CalcTokens(content)},
		"delta": map[string]interface{}{
			"stop_reason":   "end_turn",
			"stop_sequence": nil,
		},
	})
	time.Sleep(time.Second)
	response.Event(ctx, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})
	return
}

func mergeMessages(ctx *gin.Context, completion pkg.ChatCompletion) (fileMessage string, text string, tokens int, err error) {
	text = notice
	{
		values, ok := common.GetGinValue[[]pkg.Keyv[interface{}]](ctx, vars.GinClaudeMessages)
		if ok {
			var contents []string
			for _, message := range values {
				contents = append(contents, message.GetString("content"))
			}

			fileMessage = strings.Join(contents, "\n\n")
			tokens += common.CalcTokens(fileMessage)
			return
		}
	}

	var (
		messages = completion.Messages

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
			user = "Human: "
		}
		if assistant == "" {
			assistant = "Assistant: "
		}
	}

	cond := func(expr string) string {
		switch expr {
		case "assistant", "end":
			return expr
		default:
			return "user"
		}
	}

	// 合并历史对话
	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (result []map[string]string, err error) {
		role := opts.Message["role"]
		if cond(role) == cond(opts.Next) {
			// cache buffer
			if role == "function" || role == "tool" {
				opts.Buffer.WriteString(fmt.Sprintf("这是内置工具的返回结果: (%s)\n\n##\n%s\n##", opts.Message["name"], opts.Message["content"]))
				return
			}

			prefix := ""
			if role == "user" && len(opts.Message["content"]) > 0 {
				if !strings.HasPrefix(opts.Message["content"], "Assistant:") {
					prefix = user
				}
			}
			opts.Buffer.WriteString(prefix + opts.Message["content"])
			return
		}

		defer opts.Buffer.Reset()
		prefix := ""
		if role == "user" && len(opts.Message["content"]) > 0 {
			if !strings.HasPrefix(opts.Message["content"], "Assistant:") {
				prefix = user
			}
		}

		opts.Buffer.WriteString(prefix + opts.Message["content"])
		result = append(result, map[string]string{
			"role":    cond(role),
			"content": opts.Buffer.String(),
		})
		return
	}

	newMessages, err := common.TextMessageCombiner(messages, iterator)
	if err != nil {
		err = logger.WarpError(err)
		return
	}

	// 理论上合并后的上下文不存在相邻的相同消息
	var contents []string
	pos := 0
	messageL := len(newMessages)
	for {
		if pos > messageL-1 {
			break
		}

		message := newMessages[pos]
		contents = append(contents, message["content"])
		pos++
	}

	fileMessage = strings.Join(contents, "\n\n")
	tokens += common.CalcTokens(fileMessage)
	return
}
