package lmsys

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func completeToolCalls(ctx *gin.Context, proxies string, req gpt.ChatCompletionRequest) (bool, error) {
	logrus.Infof("completeTools ...")
	prompt, err := middle.BuildToolCallsTemplate(
		req.Tools,
		req.Messages,
		agent.CQConditions, 5)
	if err != nil {
		return false, err
	}

	ch, err := fetch(ctx, proxies, prompt, options{
		model:       req.Model,
		temperature: req.Temperature,
		topP:        req.TopP,
		maxTokens:   req.MaxTokens,
	})
	if err != nil {
		return false, err
	}

	content, err := waitMessage(ch)
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
	return parseToToolCall(ctx, proxies, fun, req)
}

func parseToToolCall(ctx *gin.Context, proxies string, fun *gpt.Function, req gpt.ChatCompletionRequest) (bool, error) {
	logrus.Infof("parseToToolCall ...")
	prompt, err := middle.BuildToolCallsTemplate(
		[]struct {
			Fun gpt.Function `json:"function"`
			T   string       `json:"type"`
		}{{Fun: *fun, T: "function"}},
		req.Messages,
		agent.ExtractJson, 5)
	if err != nil {
		return false, err
	}

	ch, err := fetch(ctx, proxies, prompt, options{
		model:       req.Model,
		temperature: req.Temperature,
		topP:        req.TopP,
		maxTokens:   req.MaxTokens,
	})
	if err != nil {
		return false, err
	}

	content, err := waitMessage(ch)
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
		if req.Stream {
			middle.ResponseWithSSE(ctx, MODEL, content, nil, created)
			return false, nil
		} else {
			middle.ResponseWith(ctx, MODEL, content)
			return false, nil
		}
	}

	if req.Stream {
		middle.ResponseWithSSEToolCalls(ctx, MODEL, fun.Name, argv, created)
		return false, nil
	} else {
		middle.ResponseWithToolCalls(ctx, MODEL, fun.Name, argv)
		return false, nil
	}
}
