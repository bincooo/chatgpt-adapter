package middle

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func buildTemplate(tools []pkg.Keyv[interface{}], messages []pkg.Keyv[interface{}], template string, max int) (message string, err error) {
	pMessages := messages
	content := "continue"
	if messageL := len(messages); messageL > 0 && messages[messageL-1]["role"] == "user" {
		content = messages[messageL-1].GetString("content")
		if max == 0 {
			pMessages = make([]pkg.Keyv[interface{}], 0)
		} else if max > 0 && messageL > max {
			pMessages = messages[messageL-max : messageL-1]
		} else {
			pMessages = messages[:messageL-1]
		}
	}

	for _, t := range tools {
		if !t.GetKeyv("function").Has("id") {
			t.GetKeyv("function").Set("id", common.RandStr(5))
		}
	}

	parser := templateBuilder().
		Vars("tools", tools).
		Vars("pMessages", pMessages).
		Vars("content", content).
		Do()
	return parser(template)
}

// 执行工具选择器
//
//	return:
//		bool  > 是否执行了工具
//		error > 执行异常
func CompleteToolCalls(ctx *gin.Context, req pkg.ChatCompletion, callback func(message string) (string, error)) (bool, error) {
	message, err := buildTemplate(
		req.Tools,
		req.Messages,
		agent.ToolCall, 5)
	if err != nil {
		return false, err
	}

	content, err := callback(message)
	if err != nil {
		return false, err
	}
	logrus.Infof("completeTools response: \n%s", content)

	// 解析参数
	return parseToToolCall(ctx, content, req), nil
}

func parseToToolCall(ctx *gin.Context, content string, req pkg.ChatCompletion) bool {
	created := time.Now().Unix()
	left := strings.Index(content, "{")
	right := strings.LastIndex(content, "}")
	argv := ""
	if left >= 0 && right > left {
		argv = content[left : right+1]
	} else {
		// 没有解析出 JSON
		return false
	}

	name := ""
	for _, t := range req.Tools {
		if strings.Contains(argv, t.GetKeyv("function").GetString("id")) {
			name = t.GetKeyv("function").GetString("name")
			break
		}
	}

	// 没有匹配到工具
	if name == "" {
		return false
	}

	if req.Stream {
		SSEToolCallResponse(ctx, req.Model, name, argv, created)
		return true
	} else {
		ToolCallResponse(ctx, req.Model, name, argv)
		return true
	}
}

func ToolCallCancel(str string) bool {
	return len(str) > 1 && !strings.HasPrefix(str, "1:")
}
