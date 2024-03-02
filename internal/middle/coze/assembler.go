package coze

import (
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const MODEL = "coze"

var (
	botId   = "7339624035606904840"
	version = "1709391847426"
	scene   = 2
)

func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest) {
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)

	options := coze.NewDefaultOptions(botId, version, scene, proxies)

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

	pMessages, err := buildConversation(messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	msToken := ""
	if !strings.Contains(cookie, "[msToken=") {
		middle.ResponseWithV(ctx, -1, "please provide the '[msToken=xxx]' cookie parameter")
		return
	} else {
		co := strings.Split(cookie, "[msToken=")
		msToken = strings.TrimSuffix(co[1], "]")
		cookie = co[0]
	}

	chat := coze.New(cookie, msToken, options)
	chatResponse, err := chat.Reply(ctx.Request.Context(), pMessages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	waitResponse(ctx, chatResponse, req.Stream)
}

func Generation(ctx *gin.Context, req gpt.ChatGenerationRequest) {
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)

	options := coze.NewDefaultOptions("7338032064396214278", "1708525794004", 2, proxies)
	msToken := ""

	if !strings.Contains(cookie, "[msToken=") {
		middle.ResponseWithV(ctx, -1, "please provide the '[msToken=xxx]' cookie parameter")
		return
	} else {
		co := strings.Split(cookie, "[msToken=")
		msToken = strings.TrimSuffix(co[1], "]")
		cookie = co[0]
	}

	chat := coze.New(cookie, msToken, options)
	image, err := chat.Images(ctx.Request.Context(), req.Prompt)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"data": []map[string]string{
			{"url": image},
		},
	})
}

func completeToolCalls(ctx *gin.Context, cookie, proxies string, req gpt.ChatCompletionRequest) (bool, error) {
	logrus.Infof("completeTools ...")
	toolsMap, prompt, err := middle.BuildToolCallsTemplate(
		req.Tools,
		req.Messages,
		agent.BingToolCallsTemplate, 5)
	if err != nil {
		return false, err
	}

	options := coze.NewDefaultOptions("7339624035606904840", "1708909262893", 2, proxies)

	msToken := ""
	if !strings.Contains(cookie, "[msToken=") {
		return false, errors.New("please provide the '[msToken=xxx]' cookie parameter")
	} else {
		co := strings.Split(cookie, "[msToken=")
		msToken = strings.TrimSuffix(co[1], "]")
		cookie = co[0]
	}

	chat := coze.New(cookie, msToken, options)
	chatResponse, err := chat.Reply(ctx.Request.Context(), []coze.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	})
	if err != nil {
		return false, err
	}

	content, err := waitMessage(chatResponse)
	if err != nil {
		return false, err
	}
	logrus.Infof("completeTools response: \n%s", content)
	return parseToToolCall(ctx, toolsMap, content, req.Stream)
}

func parseToToolCall(ctx *gin.Context, toolsMap map[string]string, content string, sse bool) (bool, error) {
	created := time.Now().Unix()
	for k, v := range toolsMap {
		if strings.Contains(content, k) {
			left := strings.Index(content, "{")
			right := strings.LastIndex(content, "}")
			argv := ""
			if left >= 0 && right > left {
				argv = content[left : right+1]
			}

			if sse {
				middle.ResponseWithSSEToolCalls(ctx, MODEL, v, argv, created)
				return false, nil
			} else {
				middle.ResponseWithToolCalls(ctx, MODEL, v, argv)
				return false, nil
			}
		}
	}
	return true, nil
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

func waitResponse(ctx *gin.Context, chatResponse chan string, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if strings.HasPrefix(message, "error: ") {
			middle.ResponseWithV(ctx, -1, strings.TrimPrefix(message, "error: "))
			return
		}

		message = strings.TrimPrefix(message, "text: ")
		contentL := len(message)
		if contentL <= 0 {
			continue
		}

		fmt.Printf("----- raw -----\n %s\n", message)
		if sse {
			middle.ResponseWithSSE(ctx, MODEL, message, created)
		} else {
			content += message
		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", created)
	}
}

func buildConversation(messages []map[string]string) (pMessages []coze.Message, err error) {
	var prompt string
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	if messages[pos]["role"] == "function" {
		prompt = "继续输出"
		if pos-1 >= 0 { // 获取上一条记录
			if msg := messages[pos-1]; msg["role"] == "user" {
				prompt = msg["content"]
			}
		}
	} else if messages[pos]["role"] != "user" {
		c := []rune(messages[pos]["content"])
		if contentL := len(c); contentL > 10 {
			prompt = fmt.Sprintf("从`%s`断点处继续写", string(c[contentL-10:]))
		} else {
			prompt = "继续输出"
		}
	}

	if len(prompt) > 0 {
		messages = append(messages, map[string]string{
			"role":    "user",
			"content": prompt,
		})
	}

	pos = 0
	messageL := len(messages)

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
				pMessages = append(pMessages, coze.Message{
					Role:    role,
					Content: strings.Join(buffer, "\n\n"),
				})
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return nil, errors.New(
				fmt.Sprintf("'%s' is not one of ['system', 'assistant', 'user', 'function'] - 'messages.%d.role'",
					message["role"], pos))
		}

		pos++
		if role == "" {
			role = curr
		}

		if curr == "function" {
			content = fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], content)
		}

		if curr == role {
			buffer = append(buffer, content)
			continue
		}
		pMessages = append(pMessages, coze.Message{
			Role:    role,
			Content: strings.Join(buffer, "\n\n"),
		})
		buffer = append(make([]string, 0), content)
		role = curr
	}

	return pMessages, nil
}
