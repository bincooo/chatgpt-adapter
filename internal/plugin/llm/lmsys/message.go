package lmsys

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

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan string, cancel chan error, sse bool) (content string) {
	created := time.Now().Unix()
	logger.Info("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)

	for {
		select {
		case err := <-cancel:
			if err != nil {
				if response.NotSSEHeader(ctx) {
					logger.Error(err)
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

			raw = common.ExecMatchers(matchers, raw)
			if len(raw) == 0 {
				continue
			}

			if sse && len(raw) > 0 {
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

func mergeMessages(ctx *gin.Context, messages []pkg.Keyv[interface{}]) (newMessages string) {
	{
		values, ok := common.GetGinValue[[]pkg.Keyv[interface{}]](ctx, vars.GinClaudeMessages)
		if ok {
			var contents []string
			for _, message := range values {
				contents = append(contents, message.GetString("content"))
			}
			newMessages = strings.Join(contents, "\n\n")
			return
		}
	}

	condition := func(expr string) string {
		switch expr {
		case "system", "tool", "function", "assistant", "end":
			return expr
		default:
			return "user"
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
	}) (messages []string, _ error) {
		role := opts.Message["role"]
		if condition(role) == condition(opts.Next) {
			// cache buffer
			if role == "function" || role == "tool" {
				opts.Buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", opts.Message["name"], opts.Message["content"]))
				return
			}

			opts.Buffer.WriteString(fmt.Sprintf("%s\n%s\n<|end|>", tor(role), opts.Message["content"]))
			return
		}

		defer opts.Buffer.Reset()
		var result []string
		if opts.Previous == "system" {
			result = append(result, fmt.Sprintf("<|system|>\n%s\n<|end|>", opts.Buffer.String()))
			result = append(result, "<|assistant|>ok ~<|end|>\n")
			opts.Buffer.Reset()
		}

		opts.Buffer.WriteString(fmt.Sprintf("%s\n%s\n<|end|>", tor(role), opts.Message["content"]))
		messages = append(result, opts.Buffer.String())
		return
	}

	slices, _ := common.TextMessageCombiner(messages, iterator)
	newMessages = strings.Join(slices, "\n\n")
	newMessages += "\n" + tor("assistant")
	return
}
