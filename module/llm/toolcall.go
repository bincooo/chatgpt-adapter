package llm

import (
	"adapter/module/cache"
	"adapter/module/common"
	"adapter/module/env"
	"adapter/module/fiber/context"
	"adapter/module/fiber/model"
	"adapter/module/logger"
	"chatgpt-adapter/core/common/agent"
	"fmt"
	"github.com/gin-gonic/gin"
	"strings"
)

func WillToolExecute(ctx *context.Ctx) bool {
	// 是否开启全局自定义工具回调
	enabled := env.Env.GetBool("toolCall.enabled")
	if !enabled {
		return false
	}

	completion, ok := model.GetValue[string, *model.CompletionEntity](ctx.Record, "completion")
	if !ok {
		logger.Sugar().Warn("completion not found")
		return false
	}

	messageL := len(completion.Messages)
	if messageL == 0 {
		return false
	}

	// 检查工具链
	if len(completion.Tools) == 0 {
		return false
	}

	// 最后一轮上下文如果是function或者tool，将进行工具选择
	role := completion.Messages[messageL-1]["role"]
	return role != "function" && role != "tool"
}

// 进行工具选择
func ToolExecuted(ctx *context.Ctx, completion model.CompletionEntity, yield func(message string) (string, error)) (ok bool, err error) {
	cacheManager := cache.GetToolTasksCacheManager()
	ctx.Put("exclude-task-contents", "")
	defer logger.Sugar().Info("ToolExecuted called")

	// TODO -
	return
}

// // ========================================================================  // //

// 拆解任务, 组装任务提示并返回上下文 (包含缓存已执行的任务逻辑)
func taskComplete(ctx *context.Ctx, completion model.CompletionEntity, yield func(message string) (string, error)) (messages []model.CompletionMessageEntity, hasTasks bool) {
	cacheManager := cache.GetToolTasksCacheManager()
	messages = completion.Messages
	message, err := buildTemplate(ctx, completion, agent.ToolTasks)
	if err != nil {
		logger.Sugar().Errorf("tool tasks complete err: %v", err)
		return
	}

	toolCache := hex(completion)
	logger.Sugar().Infof("complete tasks calc hash - %s", toolCache)
	tasks, err := cacheManager.GetValue(toolCache)
	if err != nil {
		logger.Sugar().Errorf("tool cache getting err: %s", err)
		return
	}

	if tasks != nil {
		excludeTasks(completion, tasks)
		logger.Sugar().Infof("complete tasks response: <cached> %s", tasks)
		// 刷新缓存时间
		if err = cacheManager.SetValue(toolCache, tasks); err != nil {
			logger.Sugar().Errorf("tool cache setting err: %v", err)
			return
		}
	} else {
		content, completeErr := yield(message)
		if completeErr != nil {
			logger.Sugar().Errorf("complete tasks err: %v", completeErr)
			return
		}
		logger.Sugar().Infof("complete tasks response: \n%s", content)

		// 解析参数
		tasks = parse2Tasks(content, completion)
		if len(tasks) == 0 {
			return
		}

		excludeTasks(completion, tasks)
		// 刷新缓存时间
		if err = cacheManager.SetValue(toolCache, tasks); err != nil {
			logger.Sugar().Errorf("tool cache reflush expir err: %v", err)
			return
		}
	}

	// 任务提示组装
	var excTasks []string
	var contents []string
	for pos := range tasks {
		task := tasks[pos]
		toolId := common.IgnoreBoolean[string](model.GetValue[string, string](task, "toolId"))
		if task.ValueEqual("exclude", "true") {
			excTasks = append(excTasks, fmt.Sprintf("工具[%s]%s已执行", toolIdWithTools(toolId, completion.Tools), task.GetString("task")))
		} else {
			contents = append(contents, common.IgnoreBoolean[string](model.GetValue[string, string](task, "task"))+
				"。 工具推荐： toolId = "+toolIdWithTools(toolId, completion.Tools))
		}
	}

	if len(contents) == 0 {
		return messages, false
	}

	hasTasks = true
	logger.Sugar().Infof("complete exclude tasks: %s", excTasks)
	logger.Sugar().Infof("complete next task: %s", contents[0])
	ctx.Put("exclude-task-contents", strings.Join(excTasks, "，"))

	// 拼接任务信息
	for pos := len(messages) - 1; pos > 0; pos-- {
		if messages[pos].ValueEqual("role", "user") {
			messages = append(messages[:pos], messages[pos+1:]...)
			break
		}
	}
	messages = append(messages, model.CompletionMessageEntity{
		"role": "user", "content": contents[0],
	})

	return
}
