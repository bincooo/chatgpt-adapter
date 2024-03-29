package coze

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, req gpt.ChatCompletionRequest) (bool, error) {
	var notebook = ctx.GetBool("notebook")
	logrus.Infof("completeTools ...")
	prompt, err := middle.BuildToolCallsTemplate(
		req.Tools,
		req.Messages,
		agent.CQConditions, 5)
	if err != nil {
		return false, err
	}

	pMessages := []coze.Message{
		{
			Role:    "system",
			Content: prompt,
		},
	}

	options := newOptions(proxies, pMessages)
	chat := coze.New(cookie, options)

	query := ""
	if notebook && len(pMessages) > 0 {
		// notebook 模式只取第一条 content
		query = pMessages[0].Content
	} else {
		query = coze.MergeMessages(pMessages)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), query)
	if err != nil {
		return false, err
	}

	content, err := waitMessage(chatResponse)
	if err != nil {
		return false, err
	}
	logrus.Infof("completeTools response: \n%s", content)

	var fun *gpt.Function
	for _, t := range req.Tools {
		if strings.Contains(content, t.Fun.Id) {
			fun = &t.Fun
			break
		}
	}

	// 不是工具调用
	if fun == nil {
		return true, nil
	}

	// 收集参数
	return parseToToolCall(ctx, cookie, proxies, fun, req.Messages, req.Stream)
}

func parseToToolCall(ctx *gin.Context, cookie, proxies string, fun *gpt.Function, messages []map[string]string, sse bool) (bool, error) {
	var notebook = ctx.GetBool("notebook")
	logrus.Infof("parseToToolCall ...")
	prompt, err := middle.BuildToolCallsTemplate(
		[]struct {
			Fun gpt.Function `json:"function"`
			T   string       `json:"type"`
		}{{Fun: *fun, T: "function"}},
		messages,
		agent.ExtractJson, 5)
	if err != nil {
		return false, err
	}

	pMessages := []coze.Message{
		{
			Role:    "system",
			Content: prompt,
		},
	}

	options := newOptions(proxies, pMessages)
	chat := coze.New(cookie, options)

	query := ""
	if notebook && len(pMessages) > 0 {
		// notebook 模式只取第一条 content
		query = pMessages[0].Content
	} else {
		query = coze.MergeMessages(pMessages)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), query)
	if err != nil {
		return false, err
	}

	content, err := waitMessage(chatResponse)
	if err != nil {
		return false, err
	}
	logrus.Infof("parseToToolCall response: \n%s", content)

	created := time.Now().Unix()
	left := strings.Index(content, "{")
	right := strings.LastIndex(content, "}")
	argv := ""
	if left >= 0 && right > left {
		argv = content[left : right+1]
	} else {
		// 没有解析出 JSON
		if sse {
			middle.ResponseWithSSE(ctx, MODEL, content, created)
			return false, nil
		} else {
			middle.ResponseWith(ctx, MODEL, content)
			return false, nil
		}
	}

	if sse {
		middle.ResponseWithSSEToolCalls(ctx, MODEL, fun.Name, argv, created)
		return false, nil
	} else {
		middle.ResponseWithToolCalls(ctx, MODEL, fun.Name, argv)
		return false, nil
	}
}
