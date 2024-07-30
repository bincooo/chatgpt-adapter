package vecmul

import (
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"fmt"
	"github.com/bincooo/vecmul.com"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func waitMessage(data chan vecmul.Data, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-data
		if !ok {
			break
		}

		if message.Error != nil {
			return "", logger.WarpError(message.Error)
		}

		if len(message.Content) > 0 {
			content += message.Content
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, cancel chan error, data chan vecmul.Data, sse bool) (content string) {
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
			message, ok := <-data
			if !ok {
				goto label
			}

			if message.Error != nil {
				logger.Error(message.Error)
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, message.Error)
				}
				return
			}

			var raw string
			raw = message.Content
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

func mergeMessages(ctx *gin.Context, completion pkg.ChatCompletion) (message string, tokens int, err error) {
	values, ok := common.GetGinValue[[]pkg.Keyv[interface{}]](ctx, vars.GinClaudeMessages)
	if ok {
		var contents []string
		for _, mes := range values {
			contents = append(contents, mes.GetString("content"))
		}

		message = strings.Join(contents, "\n\n")
		tokens += common.CalcTokens(message)
		return
	}

	var messages = completion.Messages
	condition := func(expr string) string {
		switch expr {
		case "user", "function", "tool":
			return "user"
		case "system", "assistant", "end":
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

	// 合并历史对话
	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (result []string, err error) {
		role := opts.Message["role"]
		tokens += common.CalcTokens(opts.Message["content"])
		if condition(role) == condition(opts.Next) {
			// cache buffer
			if role == "function" || role == "tool" {
				opts.Buffer.WriteString(fmt.Sprintf("这是内置工具的返回结果: (%s)\n\n##\n%s\n##", opts.Message["name"], opts.Message["content"]))
				return
			}

			opts.Buffer.WriteString(fmt.Sprintf("%s\n%s\n<|end|>", tor(condition(role)), opts.Message["content"]))
			return
		}

		defer opts.Buffer.Reset()
		opts.Buffer.WriteString(fmt.Sprintf("%s\n%s\n<|end|>", tor(condition(role)), opts.Message["content"]))
		result = append(result, opts.Buffer.String())
		return
	}
	newMessages, err := common.TextMessageCombiner(messages, iterator)
	if err != nil {
		err = logger.WarpError(err)
		return
	}

	message = strings.Join(newMessages, "\n\n")
	return
}
