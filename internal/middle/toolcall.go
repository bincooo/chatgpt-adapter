package middle

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func buildTemplate(ctx *gin.Context, completion pkg.ChatCompletion, template string, max int) (message string, err error) {
	toolDef := ctx.GetString("tool")
	if toolDef == "" {
		toolDef = "-1"
	}

	pMessages := completion.Messages
	content := "continue"
	if messageL := len(pMessages); messageL > 0 && pMessages[messageL-1]["role"] == "user" {
		content = pMessages[messageL-1].GetString("content")
		if max == 0 {
			pMessages = make([]pkg.Keyv[interface{}], 0)
		} else if max > 0 && messageL > max {
			pMessages = pMessages[messageL-max : messageL-1]
		} else {
			pMessages = pMessages[:messageL-1]
		}
	}

	for _, t := range completion.Tools {
		id := common.RandStr(5)
		fn := t.GetKeyv("function")
		if !fn.Has("id") {
			t.GetKeyv("function").Set("id", id)
		} else {
			id = fn.GetString("id")
		}

		if toolDef != "-1" && fn.Has("name") {
			if toolDef == fn.GetString("name") {
				toolDef = id
			}
		}
	}

	parser := templateBuilder().
		Vars("toolDef", toolDef).
		Vars("tools", completion.Tools).
		Vars("pMessages", pMessages).
		Vars("content", content).
		Func("join", func(slice []interface{}, sep string) string {
			if len(slice) == 0 {
				return ""
			}
			var result []string
			for _, v := range slice {
				result = append(result, fmt.Sprintf("\"%v\"", v))
			}
			return strings.Join(result, sep)
		}).Do()
	return parser(template)
}

// 执行工具选择器
//
//	return:
//		bool  > 是否执行了工具
//		error > 执行异常
func CompleteToolCalls(ctx *gin.Context, completion pkg.ChatCompletion, callback func(message string) (string, error)) (bool, error) {
	message, err := buildTemplate(ctx, completion, agent.ToolCall, 5)
	if err != nil {
		return false, err
	}

	content, err := callback(message)
	if err != nil {
		return false, err
	}
	logrus.Infof("completeTools response: \n%s", content)

	previousTokens := common.CalcTokens(message)
	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, previousTokens))

	// 解析参数
	return parseToToolCall(ctx, content, completion), nil
}

func parseToToolCall(ctx *gin.Context, content string, completion pkg.ChatCompletion) bool {
	j := ""
	created := time.Now().Unix()
	slice := strings.Split(content, "TOOL_RESPONSE")

	for _, value := range slice {
		left := strings.Index(value, "{")
		right := strings.LastIndex(value, "}")
		if left >= 0 && right > left {
			j = value[left : right+1]
			break
		}
	}

	if j == "" {
		// 没有解析出 JSON
		return false
	}

	name := ""
	for _, t := range completion.Tools {
		if strings.Contains(j, t.GetKeyv("function").GetString("id")) {
			name = t.GetKeyv("function").GetString("name")
			break
		}
	}

	// 没有匹配到工具
	if name == "" {
		return false
	}

	// 解析参数
	var js map[string]interface{}
	if err := json.Unmarshal([]byte(j), &js); err != nil {
		logrus.Error(err)
		return false
	}

	obj, ok := js["arguments"]
	if !ok {
		delete(js, "toolId")
		obj = js
	}
	bytes, _ := json.Marshal(obj)

	if completion.Stream {
		SSEToolCallResponse(ctx, completion.Model, name, string(bytes), created)
		return true
	} else {
		ToolCallResponse(ctx, completion.Model, name, string(bytes))
		return true
	}
}

func ToolCallCancel(str string) bool {
	if strings.Contains(str, "<|tool|>") {
		return true
	}
	if strings.Contains(str, "<|assistant|>") {
		return true
	}
	if strings.Contains(str, "<|user|>") {
		return true
	}
	if strings.Contains(str, "<|system|>") {
		return true
	}
	if strings.Contains(str, "<|end|>") {
		return true
	}
	return len(str) > 1 && !strings.HasPrefix(str, "1:")
}
