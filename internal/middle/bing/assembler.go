package bing

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

const MODEL = "bing"
const sysPrompt = "This is the conversation record and description stored locally as \"JSON\" : (\" System \"is the system information,\" User \"is the user message,\" Function \"is the execution result of the built-in tool, and\" Assistant \"is the reply information of the system assistant)"

func Complete(ctx *gin.Context, cookie, proxies string, req gpt.ChatCompletionRequest) {
	options, err := edge.NewDefaultOptions(cookie, "")
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}

	messages := req.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, "[] is too short - 'messages'")
		return
	}

	if messages[messageL-1]["role"] != "function" && len(req.Tools) > 0 {
		goOn, e := completeToolCalls(ctx, cookie, proxies, req)
		if e != nil {
			middle.ResponseWithE(ctx, e)
			return
		}
		if !goOn {
			return
		}
	}

	pMessages, prompt, err := buildConversation(messages)
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}

	chat := edge.New(options.
		Proxies(proxies).
		TopicToE(true).
		Model(edge.ModelSydney).
		Temperature(req.Temperature))

	chatResponse, err := chat.Reply(ctx.Request.Context(), prompt, nil, pMessages)
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}
	defer func() {
		go chat.Delete()
	}()
	waitResponse(ctx, chatResponse, req.Stream)
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

	options, err := edge.NewDefaultOptions(cookie, "")
	if err != nil {
		return false, err
	}

	chat := edge.New(options.
		Proxies(proxies).
		TopicToE(true).
		Notebook(true).
		Model(edge.ModelCreative).
		Temperature(req.Temperature))
	chatResponse, err := chat.Reply(ctx.Request.Context(), prompt, nil, nil)
	if err != nil {
		return false, err
	}

	defer func() {
		go chat.Delete()
	}()
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

func waitMessage(chatResponse chan edge.ChatResponse) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			return "", message.Error.Message
		}

		if len(message.Text) > 0 {
			content = message.Text
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, chatResponse chan edge.ChatResponse, sse bool) {
	pos := 0
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			middle.ResponseWithE(ctx, message.Error)
			return
		}

		contentL := len(message.Text)
		if pos < contentL {
			content = message.Text[pos:contentL]
			fmt.Printf("----- raw -----\n %s\n", content)
		}
		pos = contentL

		if sse {
			middle.ResponseWithSSE(ctx, MODEL, content, created)
		} else if len(message.Text) > 0 {
			content = message.Text
		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", created)
	}
}

func buildConversation(messages []map[string]string) (pMessages []edge.ChatMessage, prompt string, err error) {
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	if messages[pos]["role"] == "user" {
		prompt = messages[pos]["content"]
		messages = messages[:pos]
	} else if messages[pos]["role"] == "function" {
		prompt = "继续输出"
		if pos-1 >= 0 { // 获取上一条记录
			if msg := messages[pos-1]; msg["role"] == "user" {
				prompt = msg["content"]
			}
		}
	} else {
		c := []rune(messages[pos]["content"])
		if contentL := len(c); contentL > 10 {
			prompt = fmt.Sprintf("从`%s`断点处继续写", string(c[contentL-10:]))
		} else {
			prompt = "继续输出"
		}
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

	pMessagesVar := make([]map[string]string, 0)

	// 区块
	blockProcessing := func(title string, buf []string) map[string]string {
		content := strings.Join(buf, "\n\n")
		dict := make(map[string]string)
		dict["sender"] = title
		dict["content"] = content
		return dict
	}

	// 合并历史对话
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessagesVar = append(pMessagesVar, blockProcessing(strings.Title(role), buffer))
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return nil, "", errors.New(
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
		pMessagesVar = append(pMessagesVar, blockProcessing(strings.Title(role), buffer))
		buffer = append(make([]string, 0), content)
		role = curr
	}

	if len(pMessagesVar) > 0 {
		dict := make(map[string]interface{})
		dict["id"] = uuid.NewString()
		dict["language"] = "zh"
		dict["system_prompt"] = sysPrompt
		dict["participants"] = []string{"System", "Function", "Assistant", "User"}
		dict["messages"] = pMessagesVar
		indent, e := json.MarshalIndent(dict, "", "  ")
		if e != nil {
			return nil, "", e
		}
		pMessages = append(pMessages, edge.ChatMessage{
			"author":      "user",
			"privacy":     "Internal",
			"description": string(indent),
			"contextType": "WebPage",
			"messageType": "Context",
			"sourceName":  "history.json",
			"sourceUrl":   "file:///history.json",
			"messageId":   "discover-web--page-ping-mriduna-----",
		})
	}

	return pMessages, prompt, nil
}
