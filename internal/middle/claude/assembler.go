package claude

import (
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	claude2 "github.com/bincooo/claude-api"
	"github.com/bincooo/claude-api/types"
	"github.com/bincooo/claude-api/vars"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"time"
)

const MODEL = "claude-2"
const padtxtMaxCount = 25000

func Complete(ctx *gin.Context, cookie, proxies string, chatCompletionRequest gpt.ChatCompletionRequest) {
	options := claude2.NewDefaultOptions(cookie, vars.Model4WebClaude2)
	options.Agency = proxies

	messages := chatCompletionRequest.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, "[] is too short - 'messages'")
		return
	}

	if messages[messageL-1]["role"] != "function" && len(chatCompletionRequest.Tools) > 0 {
		goOn, _err := completeToolCalls(ctx, cookie, proxies, chatCompletionRequest)
		if _err != nil {
			middle.ResponseWithE(ctx, _err)
			return
		}
		if !goOn {
			return
		}
	}

	attr, err := buildConversation(messages)
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}

	chat, err := claude2.New(options)
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), "", attr)
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}
	defer chat.Delete()
	waitResponse(ctx, chatResponse, chatCompletionRequest.Stream)
}

func completeToolCalls(ctx *gin.Context, cookie, proxies string, chatCompletionRequest gpt.ChatCompletionRequest) (bool, error) {
	logrus.Infof("completeTools ...")
	toolsMap, prompt, err := middle.BuildToolCallsTemplate(
		chatCompletionRequest.Tools,
		chatCompletionRequest.Messages,
		agent.ClaudeToolCallsTemplate, 5)
	if err != nil {
		return false, err
	}

	options := claude2.NewDefaultOptions(cookie, vars.Model4WebClaude2)
	options.Agency = proxies

	chat, err := claude2.New(options)
	if err != nil {
		return false, err
	}

	if s := padtxt(padtxtMaxCount - len(prompt)); s != "" {
		prompt = fmt.Sprintf("%s\n--------\n\n%s", s, prompt)
	}
	chatResponse, err := chat.Reply(ctx.Request.Context(), "", []types.Attachment{
		{
			Content:  prompt,
			FileName: "paste.txt",
			FileSize: len(prompt),
			FileType: "text/plain",
		},
	})
	if err != nil {
		return false, err
	}
	defer chat.Delete()
	content, err := waitMessage(chatResponse)
	if err != nil {
		return false, err
	}
	logrus.Infof("completeTools response: \n%s", content)
	return parseToToolCall(ctx, toolsMap, content, chatCompletionRequest.Stream)
}

func parseToToolCall(ctx *gin.Context, toolsMap map[string]string, content string, sse bool) (bool, error) {
	created := time.Now().Unix()
	// 不合法标记
	if strings.Contains(content, "questionType") {
		return true, nil
	}

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

func waitMessage(chatResponse chan types.PartialResponse) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			return "", message.Error
		}

		if len(message.Text) > 0 {
			content += message.Text
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, chatResponse chan types.PartialResponse, sse bool) {
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

		fmt.Printf("----- raw -----\n %s\n", message.Text)
		if sse {
			middle.ResponseWithSSE(ctx, MODEL, message.Text, created)
		} else if len(message.Text) > 0 {
			content += message.Text
		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", created)
	}
}

func buildConversation(messages []map[string]string) (attrs []types.Attachment, err error) {
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	prompt := ""
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
		case "system", "function", "assistant":
			return expr
		case "user":
			return "human"
		default:
			return ""
		}
	}

	pMessages := ""

	// 合并历史对话
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessages += fmt.Sprintf("%s： %s", strings.Title(role), strings.Join(buffer, "\n\n"))
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
		pMessages += fmt.Sprintf("%s： %s", strings.Title(role), strings.Join(buffer, "\n\n"))
		buffer = append(make([]string, 0), content)
		role = curr
	}

	if pMessages != "" {
		if prompt != "" {
			pMessages += "Human：" + prompt
		}

		if s := padtxt(padtxtMaxCount - len(pMessages)); s != "" {
			pMessages = fmt.Sprintf("%s\n--------\n\n%s", s, pMessages)
		}

		attrs = append(attrs, types.Attachment{
			Content:  pMessages,
			FileName: "paste.txt",
			FileSize: len(pMessages),
			FileType: "text/plain",
		})
	}

	return attrs, nil
}

func padtxt(length int) string {
	if length <= 0 {
		return ""
	}

	s := "abcdefghijklmnopqrstuvwsyz0123456789!@#$%^&*()_+,.?/\\"
	bytes := make([]byte, length)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for idx := 0; idx < length; idx++ {
		pos := r.Intn(len(s))
		u := s[pos]
		bytes[idx] = u
	}
	return string(bytes)
}
