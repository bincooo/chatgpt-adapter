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
	"regexp"
	"strings"
	"time"
)

const MODEL = "cohere"

var (
	// 35-16k
	botId35_16k   = "7353052833752694791"
	version35_16k = "1712016747307"
	scene35_16k   = 2

	// 8k
	botId8k   = "7353047124357365778"
	version8k = "1712016843935"
	scene8k   = 2

	// 128k
	botId128k   = "7353048532129644562"
	version128k = "1712016880672"
	scene128k   = 2
)

func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []common.Matcher) {
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
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

	pMessages, system, message, err := buildConversation(messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	chat := cohere.New(cookie, req.Temperature, -1, req.Model)
	chat.Proxies(proxies)
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
	prefix := ""
	cmd := ctx.GetInt("cmd")

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
		if cmd >= 0 {
			if len(prefix) < 2 {
				prefix += raw
			}

			if len(prefix) < 2 {
				continue
			}

			matched, _ := regexp.MatchString("^\\d+:", prefix)
			if !matched {
				raw = fmt.Sprintf("%d: %s", cmd, prefix)
			} else {
				raw = prefix
			}
			cmd = -1
		}
		raw = common.ExecMatchers(matchers, raw)
		if sse {
			middle.ResponseWithSSE(ctx, MODEL, raw, created)
		} else {
			content += raw
		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", created)
	}
}

func buildConversation(messages []map[string]string) (pMessages []cohere.Message, system, content string, err error) {
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

	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessages = append(pMessages, cohere.Message{
					Role:    convRole(role),
					Message: strings.Join(buffer, "\n\n"),
				})
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		tMessage := message["content"]
		if curr == "" {
			return nil, "", "", errors.New(
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
		pMessages = append(pMessages, cohere.Message{
			Role:    convRole(role),
			Message: strings.Join(buffer, "\n\n"),
		})
		buffer = append(make([]string, 0), tMessage)
		role = curr
	}

	return
}
