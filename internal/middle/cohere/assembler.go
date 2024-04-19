package coh

import (
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/cohere-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const MODEL = "cohere"

func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []common.Matcher) {
	var (
		cookie   = ctx.GetString("token")
		proxies  = ctx.GetString("proxies")
		notebook = ctx.GetBool("notebook")
	)

	messages := req.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, -1, "[] is too short - 'messages'")
		return
	}

	if messages[messageL-1]["role"] != "function" && len(req.Tools) > 0 {
		goOn, e := completeToolCalls(ctx, cookie, proxies, req)
		if e != nil {
			errMessage := e.Error()
			if strings.Contains(errMessage, "Login verification is invalid") {
				middle.ResponseWithV(ctx, http.StatusUnauthorized, errMessage)
			}
			middle.ResponseWithV(ctx, -1, errMessage)
			return
		}
		if !goOn {
			return
		}
	}

	var system string
	var message string
	var pMessages []cohere.Message
	var chat cohere.Chat
	if notebook {
		m, err := buildConversation(messages)
		if err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}

		message = m
		chat = cohere.New(cookie, req.Temperature, req.Model, false)
		chat.Proxies(proxies)
		chat.TopK(req.TopK)
		chat.MaxTokens(req.MaxTokens)
		chat.StopSequences([]string{
			"user:",
			"assistant:",
			"system:",
		})
	} else {
		p, s, m, tokens, err := buildChatConversation(messages)
		if err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}

		ctx.Set("tokens", tokens)

		system = s
		message = m
		pMessages = p
		chat = cohere.New(cookie, req.Temperature, req.Model, true)
		chat.Proxies(proxies)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), pMessages, system, message)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	waitResponse(ctx, matchers, chatResponse, req.Stream)
}

func waitMessage(chatResponse chan string) (content string, err error) {

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
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan string, sse bool) {
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
			middle.ResponseWithV(ctx, -1, strings.TrimPrefix(raw, "error: "))
			return
		}

		raw = strings.TrimPrefix(raw, "text: ")
		contentL := len(raw)
		if contentL <= 0 {
			continue
		}

		fmt.Printf("----- raw -----\n %s\n", raw)
		raw = common.ExecMatchers(matchers, raw)
		if sse {
			middle.ResponseWithSSE(ctx, MODEL, raw, nil, created)
		}
		content += raw
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", common.CalcUsageTokens(content, tokens), created)
	}
}

func buildConversation(messages []map[string]string) (content string, err error) {
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	messageL := len(messages)
	pMessages := make([]map[string]string, 0)

	pos = 0
	role := ""
	buffer := make([]string, 0)

	condition := func(expr string) string {
		switch expr {
		case "system", "user", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessages = append(pMessages, map[string]string{
					"role":    role,
					"content": strings.Join(buffer, "\n\n"),
				})
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		tMessage := message["content"]
		if curr == "" {
			return "", errors.New(
				fmt.Sprintf("'%s' is not one of ['system', 'assistant', 'user', 'function'] - 'messages.%d.role'",
					message["role"], pos))
		}

		pos++
		if role == "" {
			role = curr
		}

		if curr == "function" {
			tMessage = fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], tMessage)
		}

		if curr == role {
			buffer = append(buffer, tMessage)
			continue
		}
		pMessages = append(pMessages, map[string]string{
			"role":    role,
			"content": strings.Join(buffer, "\n\n"),
		})
		buffer = append(make([]string, 0), tMessage)
		role = curr
	}

	// 尾部添加一个assistant空消息
	if pMessages[len(pMessages)-1]["role"] != "assistant" {
		pMessages = append(pMessages, map[string]string{
			"role":    "assistant",
			"content": "",
		})
	}

	return cohere.MergeMessages(pMessages), nil
}

func buildChatConversation(messages []map[string]string) (pMessages []cohere.Message, system, content string, tokens int, err error) {
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	if messages[pos]["role"] == "function" {
		content = "继续输出"
		if pos-1 >= 0 { // 获取上一条记录
			if msg := messages[pos-1]; msg["role"] == "user" {
				content = msg["content"]
			}
		}
	} else if messages[pos]["role"] != "user" {
		c := []rune(messages[pos]["content"])
		if contentL := len(c); contentL > 10 {
			content = fmt.Sprintf("从`%s`断点处继续写", string(c[contentL-10:]))
		} else {
			content = "继续输出"
		}
	}

	messageL := len(messages)
	if len(content) == 0 {
		content = messages[pos]["content"]
		messageL--
	}

	pos = 0
	if messageL > 0 && messages[pos]["role"] == "system" {
		system = messages[pos]["content"]
		pos++
	}

	role := ""
	buffer := make([]string, 0)
	var mergeMessages []cohere.Message
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	convRole := func(expr string) string {
		switch expr {
		case "assistant":
			return "Chatbot"
		default:
			return "User"
		}
	}

	// merge one
	for {
		if pos >= messageL {
			if join := strings.Join(buffer, "\n\n"); len(strings.TrimSpace(join)) > 0 {
				mergeMessages = append(mergeMessages, cohere.Message{
					Role:    convRole(role),
					Message: join,
				})
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		tMessage := message["content"]
		if curr == "" {
			return nil, "", "", -1, errors.New(
				fmt.Sprintf("'%s' is not one of ['system', 'assistant', 'user', 'function'] - 'messages.%d.role'",
					message["role"], pos))
		}

		pos++
		if role == "" {
			role = curr
		}

		if curr == "function" {
			tMessage = fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], tMessage)
		}

		if curr == role {
			tMessage = strings.TrimSpace(tMessage)
			if len(tMessage) > 0 {
				buffer = append(buffer, tMessage)
			}
			continue
		}
		mergeMessages = append(mergeMessages, cohere.Message{
			Role:    convRole(role),
			Message: strings.Join(buffer, "\n\n"),
		})
		buffer = append(make([]string, 0), tMessage)
		role = curr
	}

	messageL = len(mergeMessages)

	pos = 0
	role = ""
	buffer = make([]string, 0)

	// merge two
	for {
		if pos >= messageL {
			join := strings.Join(buffer, "\n\n")
			tokens += common.CalcTokens(join)
			pMessages = append(pMessages, cohere.Message{
				Role:    role,
				Message: join,
			})
			break
		}

		message := mergeMessages[pos]
		curr := message.Role
		tMessage := message.Message

		pos++
		if role == "" {
			role = curr
		}

		if curr == role {
			buffer = append(buffer, tMessage)
			continue
		}

		tokens += common.CalcTokens(strings.Join(buffer, ""))
		pMessages = append(pMessages, cohere.Message{
			Role:    role,
			Message: strings.Join(buffer, "\n\n"),
		})
		buffer = append(make([]string, 0), tMessage)
		role = curr
	}

	return
}
