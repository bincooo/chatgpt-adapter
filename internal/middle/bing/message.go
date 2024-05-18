package bing

import (
	"bytes"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
)

func waitMessage(chatResponse chan edge.ChatResponse, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			return "", message.Error.Message
		}

		if len(message.Text) > 0 {
			if cancel != nil && cancel(message.Text) {
				return content, nil
			}
			content = message.Text
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []pkg.Matcher, cancel chan error, chatResponse chan edge.ChatResponse, sse bool) {
	var (
		pos     = 0
		content = ""
		created = time.Now().Unix()
		tokens  = ctx.GetInt("tokens")
	)

	logrus.Info("waitResponse ...")
	for {
		select {
		case err := <-cancel:
			if err != nil {
				logrus.Error(err)
				if middle.NotSSEHeader(ctx) {
					middle.ErrResponse(ctx, -1, err)
				}
				return
			}
			goto label
		default:
			message, ok := <-chatResponse
			if !ok {
				goto label
			}

			if message.Error != nil {
				logrus.Error(message.Error)
				if middle.NotSSEHeader(ctx) {
					middle.ErrResponse(ctx, -1, message.Error)
				}
				return
			}

			var raw string
			contentL := len(message.Text)
			if pos < contentL {
				raw = message.Text[pos:contentL]
				fmt.Printf("----- raw -----\n %s\n", raw)
			}
			pos = contentL
			raw = pkg.ExecMatchers(matchers, raw)

			if sse {
				middle.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
		}
	}

label:
	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		middle.Response(ctx, Model, content)
	} else {
		middle.SSEResponse(ctx, Model, "[DONE]", created)
	}
}

func mergeMessages(pad bool, max int, messages []pkg.Keyv[interface{}]) (pMessages []edge.ChatMessage, text string, tokens int) {
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "function":
			return "user"
		case "assistant":
			return "bot"
		default:
			return ""
		}
	}

	// 合并历史对话
	newMessages := common.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []edge.ChatMessage {
		role := message["role"]
		tokens += common.CalcTokens(message["content"])
		if condition(role) == condition(next) {
			// cache buffer
			if role == "function" {
				buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], message["content"]))
				return nil
			}

			buffer.WriteString(fmt.Sprintf("<|%s|>\n%s\n<|end|>", role, message["content"]))
			return nil
		}

		defer buffer.Reset()
		var result []edge.ChatMessage
		if previous == "system" {
			result = append(result, edge.BuildUserMessage(buffer.String()))
			result = append(result, edge.BuildBotMessage("<|assistant|>ok ~<|end|>\n"))
			buffer.Reset()
		}

		buffer.WriteString(fmt.Sprintf("<|%s|>\n%s\n<|end|>", role, message["content"]))
		result = append(result, edge.BuildSwitchMessage(condition(role), buffer.String()))
		return result
	})

	// 尝试引导对话，避免道歉
	if pad {
		pMessages = []edge.ChatMessage{
			edge.BuildUserMessage("你好"),
			edge.BuildBotMessage("你好，这是必应。我可以用中文和你聊天，也可以帮你做一些有趣的事情，比如写诗，编程，创作歌曲，角色扮演等等。你想让我做什么呢？😊"),
			edge.BuildUserMessage("你能做什么"),
			edge.BuildBotMessage("我能做很多有趣和有用的事情，比如：\n\n- 和你聊天，了解你的兴趣和爱好，根据你的要求扮演一些有趣的角色或故事。\n- 从当前网页中的内容回答问题。\n- 描述你上传的图片，告诉你图片里有什么，或者画一幅你想要的图画。\n\n你想让我试试哪一项呢？😊"),
		}
		max -= 2
	}

	// 获取最后一条用户消息
	if pos := len(newMessages) - 1; newMessages[pos]["author"] == "user" {
		text = newMessages[pos]["text"]
		newMessages = newMessages[:pos]
	} else {
		text = "continue"
	}

	// 超出最大轮次改为WebPage
	if len(newMessages)/2 > max {
		message := edge.BuildPageMessage(common.StringCombiner(newMessages[:len(newMessages)-max*2], func(message edge.ChatMessage) string {
			return message["text"]
		}))
		pMessages = append(pMessages, message)
		newMessages = newMessages[len(newMessages)-max*2:]
	}

	pMessages = append(pMessages, newMessages...)
	return
}
