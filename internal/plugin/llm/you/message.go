package you

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/you.com"
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

func mergeMessages(ctx *gin.Context, completion pkg.ChatCompletion) (pMessages []you.Message, text string, tokens int, err error) {
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
			user = "Human："
		}
		if assistant == "" {
			assistant = "Assistant："
		}
		user += "\n"
		assistant += "\n"
	}

	cond := func(expr string) string {
		switch expr {
		case "assistant", "end":
			return expr
		default:
			return "user"
		}
	}

	for _, message := range messages {
		tokens += common.CalcTokens(message.GetString("content"))
	}

	is32 := tokens < 12000
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

	text = "Please review the attached prompt"

	// 获取最后一条用户消息
	okey := ""
	if is32 {
		okey = "ok ~"
		messageL := len(newMessages)
		message := newMessages[messageL-1]
		if message["role"] == "user" {
			newMessages = newMessages[:messageL-1]
			text = strings.TrimSpace(message["content"])
			messageL -= 1
		}
	}

	// 理论上合并后的上下文不存在相邻的相同消息
	pos := 0
	messageL := len(newMessages)
	for {
		if pos > messageL-1 {
			break
		}

		newMessage := you.Message{}
		message := newMessages[pos]
		if message["role"] == "user" {
			newMessage.Question = message["content"]
		} else {
			newMessage.Question = okey
		}

		pos++
		if pos > messageL-1 {
			newMessage.Answer = okey
			pMessages = append(pMessages, newMessage)
			break
		}

		message = newMessages[pos]
		if message["role"] == "assistant" {
			newMessage.Answer = assistant + message["content"]
		} else {
			newMessage.Answer = okey
			pMessages = append(pMessages, newMessage)
			newMessage = you.Message{
				Question: message["content"],
				Answer:   "",
			}
		}
		pMessages = append(pMessages, newMessage)
		pos++
	}
	return
}
