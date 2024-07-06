package bing

import (
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"fmt"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func waitMessage(chatResponse chan edge.ChatResponse, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			return "", logger.WarpError(message.Error.Message)
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

func waitResponse(ctx *gin.Context, matchers []common.Matcher, cancel chan error, chatResponse chan edge.ChatResponse, sse bool) (content string) {
	var (
		pos     = 0
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
			message, ok := <-chatResponse
			if !ok {
				goto label
			}

			if message.Error != nil {
				logger.Error(message.Error)
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, message.Error)
				}
				return
			}

			var raw string
			contentL := len(message.Text)
			if pos < contentL {
				raw = message.Text[pos:contentL]
				logger.Debug("----- raw -----")
				logger.Debug(raw)
			}
			pos = contentL
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

func mergeMessages(ctx *gin.Context, pad bool, max int, completion pkg.ChatCompletion) (pMessages []edge.ChatMessage, text string, tokens int, err error) {
	var messages = completion.Messages
	condition := func(expr string) string {
		switch expr {
		case "system", "user", "function", "tool":
			return "user"
		case "assistant":
			return "bot"
		default:
			return ""
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

	// 合并历史对话
	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (result []edge.ChatMessage, err error) {
		role := opts.Message["role"]
		tokens += common.CalcTokens(opts.Message["content"])

		// 复合消息
		if _, ok := opts.Message["multi"]; ok && role == "user" && completion.Model == Model+"-vision" {
			message := opts.Initial()
			content, e := processMultiMessage(ctx, message)
			if e != nil {
				return nil, logger.WarpError(e)
			}

			opts.Buffer.WriteString(fmt.Sprintf("%s\n%s\n<|end|>", tor(role), content))
			if condition(role) != condition(opts.Next) {
				result = append(result, edge.BuildUserMessage(opts.Buffer.String()))
				opts.Buffer.Reset()
			}
			return
		}

		if condition(role) == condition(opts.Next) {
			// cache buffer
			if role == "function" || role == "tool" {
				opts.Buffer.WriteString(fmt.Sprintf("这是内置工具的返回结果: (%s)\n\n##\n%s\n##", opts.Message["name"], opts.Message["content"]))
				return
			}

			opts.Buffer.WriteString(fmt.Sprintf("%s\n%s\n<|end|>", tor(role), opts.Message["content"]))
			return
		}

		defer opts.Buffer.Reset()
		if opts.Previous == "system" {
			result = append(result, edge.BuildUserMessage(opts.Buffer.String()))
			result = append(result, edge.BuildBotMessage("<|assistant|>ok ~<|end|>\n"))
			opts.Buffer.Reset()
		}

		opts.Buffer.WriteString(fmt.Sprintf("%s\n%s\n<|end|>", tor(role), opts.Message["content"]))
		result = append(result, edge.BuildSwitchMessage(condition(role), opts.Buffer.String()))
		return
	}
	newMessages, err := common.TextMessageCombiner(messages, iterator)
	if err != nil {
		err = logger.WarpError(err)
		return
	}

	// 尝试引导对话，避免道歉
	if pad {
		pMessages = baseMessages()
		max -= 2
	}

	// 获取最后一条用户消息
	for pos := len(newMessages) - 1; pos >= 0; pos-- {
		message := newMessages[pos]
		if message["author"] == "user" {
			newMessages = append(newMessages[:pos], newMessages[pos+1:]...)
			text = strings.TrimSpace(message["text"].(string))
			text = strings.TrimLeft(text, tor("user"))
			text = strings.TrimRight(text, "<|end|>")
			break
		}
	}

	// 超出最大轮次改为WebPage
	if len(newMessages)/2 > max {
		message := edge.BuildPageMessage(common.MergeStrMessage(newMessages[:len(newMessages)-max*2], func(message edge.ChatMessage) string {
			return message["text"].(string)
		}))
		pMessages = append(pMessages, message)
		pMessages = append(pMessages, edge.BuildMessage("CurrentWebpageContextRequest", "改为从此页面回答。"))
		newMessages = newMessages[len(newMessages)-max*2:]
	}
	pMessages = append(pMessages, newMessages...)

	//var previousMessages []string
	//for pos := range newMessages {
	//	message := newMessages[pos]
	//	previousMessages = append(previousMessages, message["text"])
	//}
	//
	//if len(previousMessages) > 0 {
	//	pMessages = append(pMessages, edge.BuildPageMessage(strings.Join(previousMessages, "\n\n")))
	//	if text != "continue" {
	//		pMessages = append(pMessages, edge.BuildMessage("CurrentWebpageContextRequest", "改为从此页面回答。"))
	//	} else {
	//		text = "改为从此页面回答。\n\n" + text
	//	}
	//}

	return
}

func processMultiMessage(ctx *gin.Context, message pkg.Keyv[interface{}]) (string, error) {
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)
	contents := make([]string, 0)
	values := message.GetSlice("content")
	if len(values) == 0 {
		return "", nil
	}
	for _, value := range values {
		var keyv pkg.Keyv[interface{}]
		keyv, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		if keyv.Is("type", "text") {
			contents = append(contents, keyv.GetString("text"))
			continue
		}

		if keyv.Is("type", "image_url") {
			o := keyv.GetKeyv("image_url")
			options, err := edge.NewDefaultOptions(cookie, "")
			if err != nil {
				return "", logger.WarpError(err)
			}

			chat := edge.New(options.Proxies(proxies).
				Model(edge.ModelSydney).
				TopicToE(true))
			chat.Client(plugin.HTTPClient)

			kb, err := chat.LoadImage(common.GetGinContext(ctx), o.GetString("url"))
			if err != nil {
				return "", logger.WarpError(err)
			}

			chat.KBlob(kb)
			partialResponse, err := chat.Reply(common.GetGinContext(ctx), "请你使用json代码块中文描述这张图片，不必说明直接输出结果", nil)
			if err != nil {
				return "", logger.WarpError(err)
			}

			content, err := waitMessage(partialResponse, nil)
			if err != nil {
				return "", logger.WarpError(err)
			}

			left := strings.Index(content, "{")
			right := strings.Index(content, "}")
			if left == -1 || left > right {
				return "", nil
			}

			imageInfo := fmt.Sprintf("*这是内置image工具的返回结果*： %s\n%s\n----", o.GetString("url"), content)
			contents = append(contents, imageInfo)
		}
	}

	if len(contents) == 0 {
		return "", nil
	}

	join := strings.Join(contents, "\n\n")
	return fmt.Sprintf("<|user|>\n%s<|end|>", join), nil
}

func baseMessages() []edge.ChatMessage {
	return []edge.ChatMessage{
		edge.BuildUserMessage("你好"),
		edge.BuildBotMessage("你好，这是必应。我可以用中文和你聊天，也可以帮你做一些有趣的事情，比如写诗，编程，创作歌曲，角色扮演等等。你想让我做什么呢？😊"),
		edge.BuildUserMessage("你能做什么"),
		edge.BuildBotMessage("我能做很多有趣和有用的事情，比如：\n\n- 和你聊天，了解你的兴趣和爱好，根据你的要求扮演一些有趣的角色或故事。\n- 从当前网页中的内容回答问题。\n- 描述你上传的图片，告诉你图片里有什么，或者画一幅你想要的图画。\n- 还可以作为工具选择助手。\n\n你想让我试试哪一项呢？😊"),
	}
}
