package coze

import (
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"errors"
	"fmt"
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
		proxies  = ctx.GetString("proxies")
		messages = common.GetGinCompletion(ctx).Messages
	)

	{
		values, ok := common.GetGinValue[[]pkg.Keyv[interface{}]](ctx, vars.GinClaudeMessages)
		if ok {
			var contents []string
			for _, message := range values {
				contents = append(contents, message.GetString("content"))
			}

			message := strings.Join(contents, "\n\n")
			tokens += common.CalcTokens(message)
			newMessages = append(newMessages, coze.Message{
				Role:    "user",
				Content: message,
			})
			return
		}
	}

	condition := func(expr string) string {
		switch expr {
		case "system", "assistant", "function", "tool", "end":
			return expr
		default:
			return "user"
		}
	}

	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (messages []coze.Message, _ error) {
		role := opts.Message["role"]
		tokens += common.CalcTokens(opts.Message["content"])
		// 复合消息
		if _, ok := opts.Message["multi"]; ok && role == "user" {
			message := opts.Initial()
			content, e := common.MergeMultiMessage(ctx.Request.Context(), proxies, message)
			if e != nil {
				return nil, e
			}
			opts.Buffer.WriteString(content)
			if condition(role) != condition(opts.Next) {
				messages = []coze.Message{
					{
						Role:    role,
						Content: opts.Buffer.String(),
					},
				}
				opts.Buffer.Reset()
			}
			return
		}

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
		messages = []coze.Message{
			{
				Role:    role,
				Content: opts.Buffer.String(),
			},
		}
		return
	}

	newMessages, err = common.TextMessageCombiner(messages, iterator)
	if err != nil {
		err = logger.WarpError(err)
	}
	return
}
