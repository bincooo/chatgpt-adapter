package bing

import (
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

var template = `我会给你几个问题类型，请参考背景知识（可能为空）和对话记录，判断我“本次问题”的类型，并返回一个问题“类型ID”和“参数JSON”:
<问题类型>
{{tools_types}}{"questionType": "其他问题", "typeId": "agex"}
</问题类型>

<背景知识>
你将作为系统API协调工具，为我分析给出的content并结合对话记录来判断是否需要执行哪些工具。
工具如下
## Tools
You can use these tools below:
{{tools_desc}}##

不要访问contnet中的链接内容
不可回复任何提示
不允许做任何解释
不可联网检索
</背景知识>

<对话记录>
{{history}}
</对话记录>


content= {{prompt}}

类型ID=
参数JSON=
---
补充类型ID以及参数JSON的内容，仅回复ID和JSON。
不需要解释任何结果！
不需要执行任何任务！
回答尽可能简洁！
`

func Complete(ctx *gin.Context, cookie, proxies string, chatCompletionRequest gpt.ChatCompletionRequest) {
	options, err := edge.NewDefaultOptions(cookie, "")
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}

	messages := chatCompletionRequest.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, "[] is too short - 'messages'")
		return
	}

	if messages[messageL-1]["role"] != "function" && len(chatCompletionRequest.Tools) > 0 {
		goOn, _err := completeTools(ctx, cookie, proxies, chatCompletionRequest)
		if _err != nil {
			middle.ResponseWithE(ctx, _err)
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
		Temperature(chatCompletionRequest.Temperature))

	chatResponse, err := chat.Reply(ctx.Request.Context(), prompt, nil, pMessages)
	if err != nil {
		middle.ResponseWithE(ctx, err)
		return
	}
	waitResponse(ctx, chatResponse, chatCompletionRequest.Stream)
}

func completeTools(ctx *gin.Context, cookie, proxies string, chatCompletionRequest gpt.ChatCompletionRequest) (bool, error) {
	toolsMap, prompt, err := buildTools(chatCompletionRequest.Tools, chatCompletionRequest.Messages)
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
		Temperature(chatCompletionRequest.Temperature))

	chatResponse, err := chat.Reply(ctx.Request.Context(), prompt, nil, nil)
	if err != nil {
		return false, err
	}

	content, err := waitMessage(chatResponse)
	if err != nil {
		return false, err
	}

	created := time.Now().Unix()
	for k, v := range toolsMap {
		if strings.Contains(content, k) {
			left := strings.Index(content, "{")
			right := strings.LastIndex(content, "}")
			args := ""
			if left >= 0 && right > left {
				args = content[left : right+1]
			}

			if chatCompletionRequest.Stream {
				middle.ResponseWithSSEToolCalls(ctx, "bing", v, args, created)
				return false, nil
			}
			ctx.JSON(http.StatusOK, gpt.ChatCompletionResponse{
				Model:   "bing",
				Created: created,
				Id:      "chatcmpl-completion",
				Object:  "chat.completion",
				Choices: []gpt.ChatCompletionResponseChoice{
					{
						Index: 0,
						Message: &struct {
							Role      string                   `json:"role"`
							Content   string                   `json:"content"`
							ToolCalls []map[string]interface{} `json:"tool_calls"`
						}{
							Role: "assistant",
							ToolCalls: []map[string]interface{}{
								{
									"id":   "call_" + middle.RandomString(5),
									"type": "function",
									"function": map[string]string{
										"name":      v,
										"arguments": args,
									},
								},
							},
						},
						FinishReason: "stop",
					},
				},
			})
			return false, nil
		}
	}
	return true, nil
}

func buildTools(
	tools []struct {
		Fun gpt.Function `json:"function"`
		T   string       `json:"type"`
	},
	messages []map[string]string,
) (toolsMap map[string]string, prompt string, err error) {
	t1 := ""
	t2 := ""
	history := ""

	toolsMap = make(map[string]string)
	for i, tool := range tools {
		if tool.T != "function" {
			continue
		}
		f := tool.Fun
		id := middle.RandomString(5)

		t1 += fmt.Sprintf("{\"questionType\": \"%s\", \"typeId\": \"%s\"}\n", f.Name, id)
		t2 += fmt.Sprintf(
			"%d. [%s] %s;\n\tparameters:\n",
			i+1,
			f.Name,
			f.Description,
		)

		if properties := f.Params.Properties; properties != nil {
			for k, v := range properties {
				value := v.(map[string]interface{})
				t2 += fmt.Sprintf("\t\t%s: {\n\t\t\ttype: %s\n\t\t\tdescription: %s\n\t\t}\n", k, value["type"], value["description"])
			}
		}

		toolsMap[id] = f.Name
	}

	pMessages, p, err := buildConversation(messages)
	if err != nil {
		return nil, "", err
	}

	toA := func(expr string) string {
		switch expr {
		case "bot":
			return "AI"
		default:
			return "Human"
		}
	}
	for _, message := range pMessages {
		history += fmt.Sprintf("%s: %s\n", toA(message["author"]), message["text"])
	}
	prompt = strings.Replace(template, "{{tools_types}}", t1, -1)
	prompt = strings.Replace(prompt, "{{tools_desc}}", t2, -1)
	prompt = strings.Replace(prompt, "{{history}}", history, -1)
	prompt = strings.Replace(prompt, "{{prompt}}", p, -1)
	return
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

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			middle.ResponseWithE(ctx, message.Error)
			return
		}

		if sse {
			contentL := len(message.Text)
			if pos < contentL {
				middle.ResponseWithSSE(ctx, "bing", message.Text[pos:contentL], created)
			}
			pos = contentL
		} else if len(message.Text) > 0 {
			content = message.Text
		}
	}

	if !sse {
		ctx.JSON(http.StatusOK, gpt.ChatCompletionResponse{
			Model:   "bing",
			Created: created,
			Id:      "chatcmpl-completion",
			Object:  "chat.completion",
			Choices: []gpt.ChatCompletionResponseChoice{
				{
					Index: 0,
					Message: &struct {
						Role      string                   `json:"role"`
						Content   string                   `json:"content"`
						ToolCalls []map[string]interface{} `json:"tool_calls"`
					}{"assistant", content, nil},
					FinishReason: "stop",
				},
			},
		})
	} else {
		middle.ResponseWithSSE(ctx, "bing", "[DONE]", created)
	}
}

func buildConversation(messages []map[string]string) (pMessages []edge.ChatMessage, prompt string, err error) {

	pos := len(messages) - 1
	if messages[pos]["role"] == "user" {
		prompt = messages[pos]["content"]
		messages = messages[:pos]
	} else {
		prompt = "continue"
	}

	pos = 0
	messageL := len(messages)

	role := ""
	buffer := make([]string, 0)

	toA := func(expr string) string {
		switch expr {
		case "system", "user", "function":
			return "user"
		case "assistant":
			return "bot"
		default:
			return ""
		}
	}

	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessages = append(pMessages, edge.ChatMessage{
					"author": role,
					"text":   strings.Join(buffer, "\n\n"),
				})
			}
			break
		}

		message := messages[pos]
		curr := toA(message["role"])
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

		if curr == role {
			if message["role"] == "function" {
				content = fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##\n---\n\n%s", message["name"], content, prompt)
			}
			buffer = append(buffer, content)
			continue
		}
		pMessages = append(pMessages, edge.ChatMessage{
			"author": role,
			"text":   strings.Join(buffer, "\n\n"),
		})
		buffer = append(make([]string, 0), content)
		role = curr
	}

	return pMessages, prompt, nil
}
