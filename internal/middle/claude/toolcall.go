package claude

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	claude2 "github.com/bincooo/claude-api"
	"github.com/bincooo/claude-api/types"
	"github.com/bincooo/claude-api/vars"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, req gpt.ChatCompletionRequest) (bool, error) {
	logrus.Infof("completeTools ...")
	prompt, err := middle.BuildToolCallsTemplate(
		req.Tools,
		req.Messages,
		agent.CQConditions, 5)
	if err != nil {
		return false, err
	}

	model := vars.Model4WebClaude2
	if strings.HasPrefix(req.Model, "claude-") {
		model = req.Model
	}

	options := claude2.NewDefaultOptions(cookie, model)
	options.Proxies = proxies

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
	return parseToToolCall(ctx, cookie, proxies, model, fun, req.Messages, req.Stream)
}

func parseToToolCall(ctx *gin.Context, cookie, proxies, model string, fun *gpt.Function, messages []map[string]string, sse bool) (bool, error) {
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

	options := claude2.NewDefaultOptions(cookie, model)
	options.Proxies = proxies

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
