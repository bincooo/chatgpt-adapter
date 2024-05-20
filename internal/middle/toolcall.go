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
	"slices"
	"strings"
	"time"
)

var (
	excludeToolNames = "__EXCLUDE_TOOL_NAMES__"
	MaxMessages      = 10
)

func NeedToToolCall(ctx *gin.Context) bool {
	var tool = "-1"
	{
		t := common.GetGinToolValue(ctx)
		tool = t.GetString("id")
		if tool == "-1" && t.Is("tasks", true) {
			tool = "tasks"
		}
	}

	completion := common.GetGinCompletion(ctx)
	messageL := len(completion.Messages)
	if messageL == 0 {
		return false
	}

	if len(completion.Tools) == 0 {
		return false
	}

	role := completion.Messages[messageL-1]["role"]
	return (role != "function" && role != "tool") || tool != "-1"
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
	if strings.Contains(str, "USER: ") {
		return true
	}
	if strings.Contains(str, "ANSWER: ") {
		return true
	}
	if strings.Contains(str, "TOOL_RESPONSE: ") {
		return true
	}
	return len(str) > 1 && !strings.HasPrefix(str, "1:")
}

// 执行工具选择器
//
//	return:
//	bool  > 是否执行了工具
//	error > 执行异常
func CompleteToolCalls(ctx *gin.Context, completion pkg.ChatCompletion, callback func(message string) (string, error)) (bool, error) {
	// 是否开启任务拆解
	if toolIsEnabled(ctx) {
		var hasTasks = false
		completion.Messages, hasTasks = completeToolTasks(ctx, completion, callback)
		if !hasTasks {
			return false, nil
		}
	}

	message, err := buildTemplate(ctx, completion, agent.ToolCall)
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

// 拆解任务, 组装任务提示并返回上下文
func completeToolTasks(ctx *gin.Context, completion pkg.ChatCompletion, callback func(message string) (string, error)) (messages []pkg.Keyv[interface{}], hasTasks bool) {
	messages = completion.Messages
	message, err := buildTemplate(ctx, completion, agent.ToolTasks)
	if err != nil {
		return
	}

	content, err := callback(message)
	if err != nil {
		return
	}
	logrus.Infof("completeTasks response: \n%s", content)

	// 解析参数
	tasks := parseToToolTasks(content, completion)
	if len(tasks) == 0 {
		return
	}

	// 任务提示组装
	var excludeTasks []string
	var contents []string
	for pos := range tasks {
		task := tasks[pos]
		if task.Is("exclude", "true") {
			toolId := task.GetString("toolId")
			excludeTasks = append(excludeTasks, fmt.Sprintf("工具%s已执行", toolId))
		} else {
			contents = append(contents, task.GetString("task"))
		}
	}

	if len(contents) == 0 {
		return messages, false
		//contents = append(contents, "continue")
	}

	hasTasks = true
	if len(excludeTasks) > 0 {
		contents = append([]string{fmt.Sprintf("(%s)", strings.Join(excludeTasks, ", "))}, contents...)
	}

	// 拼接任务信息
	if pos := len(messages) - 1; messages[pos].Is("role", "user") {
		messages[pos] = pkg.Keyv[interface{}]{
			"role": "user", "content": strings.Join(contents, "\n"),
		}
	} else {
		messages = append(messages, pkg.Keyv[interface{}]{
			"role": "user", "content": strings.Join(contents, "\n"),
		})
	}

	return
}

func buildTemplate(ctx *gin.Context, completion pkg.ChatCompletion, template string) (message string, err error) {
	pMessages := completion.Messages
	messageL := len(pMessages)
	content := "continue"

	if messageL > MaxMessages {
		pMessages = pMessages[messageL-MaxMessages:]
		messageL = len(pMessages)
	}

	if messageL > 0 {
		if keyv := pMessages[messageL-1]; keyv.Is("role", "user") {
			content = keyv.GetString("content")
			pMessages = pMessages[:messageL-1]
		}
	}

	if content == "continue" {
		ctx.Set(excludeToolNames, extractToolNames(completion.Messages))
	}

	for _, t := range completion.Tools {
		t.GetKeyv("function").Set("id", common.RandStr(5))
	}

	parser := templateBuilder().
		Vars("toolDef", toolDef(ctx, completion.Tools)).
		Vars("tools", completion.Tools).
		Vars("pMessages", pMessages).
		Vars("content", content).
		Func("Join", func(slice []interface{}, sep string) string {
			if len(slice) == 0 {
				return ""
			}
			var result []string
			for _, v := range slice {
				result = append(result, fmt.Sprintf("\"%v\"", v))
			}
			return strings.Join(result, sep)
		}).
		Func("Enc", func(value interface{}) string {
			return strings.ReplaceAll(fmt.Sprintf("%s", value), "\n", "\\n")
		}).
		Func("ToolDesc", func(value string) string {
			for _, t := range completion.Tools {
				fn := t.GetKeyv("function")
				if !fn.Has("name") {
					continue
				}
				if value == fn.GetString("name") {
					return fn.GetString("description")
				}
			}
			return ""
		}).Do()
	return parser(template)
}

// 工具参数解析
//
//	return:
//	bool  > 是否执行了工具
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

	// 非-1值则为有默认选项
	valueDef := nameWithToolDef(common.GetGinToolValue(ctx).GetString("id"), completion.Tools)

	// 没有解析出 JSONz
	if j == "" {
		if valueDef != "-1" {
			return toolCallResponse(ctx, completion, valueDef, "{}", created)
		}
		return false
	}

	name := ""
	for _, t := range completion.Tools {
		fn := t.GetKeyv("function")
		n := fn.GetString("name")
		// id 匹配
		if strings.Contains(j, fn.GetString("id")) {
			name = n
			break
		}
		// name 匹配
		if strings.Contains(j, n) {
			name = n
			break
		}
	}

	// 没有匹配到工具
	if name == "" {
		if valueDef != "-1" {
			return toolCallResponse(ctx, completion, valueDef, "{}", created)
		}
		return false
	}

	// 避免AI重复选择相同的工具
	if names, ok := common.GetGinValues[string](ctx, excludeToolNames); ok {
		if slices.Contains(names, name) {
			return valueDef != "-1" && toolCallResponse(ctx, completion, valueDef, "{}", created)
		}
	}

	// 解析参数
	var js map[string]interface{}
	if err := json.Unmarshal([]byte(j), &js); err != nil {
		logrus.Error(err)
		if valueDef != "-1" {
			return toolCallResponse(ctx, completion, valueDef, "{}", created)
		}
		return false
	}

	obj, ok := js["arguments"]
	if !ok {
		delete(js, "toolId")
		obj = js
	}

	bytes, _ := json.Marshal(obj)
	return toolCallResponse(ctx, completion, name, string(bytes), created)
}

// 解析任务
func parseToToolTasks(content string, completion pkg.ChatCompletion) (tasks []pkg.Keyv[string]) {
	j := ""
	slice := strings.Split(content, "TOOL_RESPONSE")
	for _, value := range slice {
		left := strings.Index(value, "[{")
		right := strings.LastIndex(value, "}]")
		if left >= 0 && right > left {
			j = value[left : right+2]
			break
		}
	}

	// 没有解析出 JSON
	if j == "" {
		return
	}

	// 解析参数
	var js []pkg.Keyv[string]
	if err := json.Unmarshal([]byte(j), &js); err != nil {
		logrus.Error(err)
		return
	}

	// 检查任务列表
	excludeNames := extractToolNames(completion.Messages)
	for pos := range js {
		task := js[pos]
		if !task.Has("toolId") {
			continue
		}

		toolId := nameWithToolDef(task.GetString("toolId"), completion.Tools)
		if toolId == "-1" || !task.Has("task") {
			continue
		}

		if slices.Contains(excludeNames, toolId) {
			task.Set("exclude", "true")
			task.Set("toolId", toolId)
		}

		tasks = append(tasks, task)
	}

	return
}

// 提取对话中的tool-names
func extractToolNames(messages []pkg.Keyv[interface{}]) (slice []string) {
	index := len(messages) - MaxMessages
	if index < 0 {
		index = 0
	}

	for pos := range messages[index:] {
		message := messages[index+pos]
		if message.Is("role", "tool") {
			slice = append(slice, message.GetString("name"))
		}
	}
	return
}

func toolCallResponse(ctx *gin.Context, completion pkg.ChatCompletion, name string, value string, created int64) bool {
	if completion.Stream {
		SSEToolCallResponse(ctx, completion.Model, name, value, created)
		return true
	} else {
		ToolCallResponse(ctx, completion.Model, name, value)
		return true
	}
}

func toolDef(ctx *gin.Context, tools []pkg.Keyv[interface{}]) (value string) {
	value = common.GetGinToolValue(ctx).GetString("id")
	if value == "-1" {
		return
	}

	for _, t := range tools {
		fn := t.GetKeyv("function")
		toolId := fn.GetString("id")
		if fn.Has("name") {
			if value == fn.GetString("name") {
				value = toolId
				return
			}
		}
	}
	return "-1"
}

// 工具名是否存在工具集中，"-1" 不存在，否则返回具体名字
func nameWithToolDef(name string, tools []pkg.Keyv[interface{}]) (value string) {
	value = name
	if value == "" {
		return "-1"
	}

	for _, t := range tools {
		fn := t.GetKeyv("function")
		if fn.Has("name") {
			if value == fn.GetString("name") {
				return
			}
		}

		if fn.Has("id") {
			if value == fn.GetString("id") {
				value = fn.GetString("name")
				return
			}
		}
	}

	return "-1"
}

func toolIsEnabled(ctx *gin.Context) bool {
	t := common.GetGinToolValue(ctx)
	return t.Is("tasks", true)
}
