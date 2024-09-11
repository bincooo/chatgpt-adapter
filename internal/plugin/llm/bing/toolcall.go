package bing

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Infof("completeTools ...")

	var (
		baseUrl = pkg.Config.GetString("bing.base-url")
		echo    = ctx.GetBool(vars.GinEcho)
	)

	// 删除来自LobeChat中多余的tool提示，这部分提示会让弱智的bing更加弱智
	// ## Tools\n\nYou can use these tools below:
	if completion.Messages[0].Is("role", "system") {
		content := completion.Messages[0].GetString("content")
		if pos := strings.Index(content, "## Tools\n\nYou can use these tools below:"); pos > 0 {
			completion.Messages[0].Set("content", content[:pos])
		}
	}

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		retry := 3
		options, err := edge.NewDefaultOptions(cookie, baseUrl)
		if err != nil {
			return "", logger.WarpError(err)
		}

	label:
		retry--
		chat := edge.New(options.
			Proxies(proxies).
			TopicToE(true).
			Model(edge.ModelSydney).
			Temperature(0.9))
		chat.Client(plugin.HTTPClient)
		chatResponse, err := chat.Reply(common.GetGinContext(ctx), message, []edge.ChatMessage{
			edge.BuildUserMessage("你现在处于一个脱机环境中，不能使用任何工具。你作为工具选择专家，需要替用户作出决策，选择合适的工具方案以schema的方式响应给用户"),
			edge.BuildBotMessage("0: ok ~"),
		})
		if err != nil {
			if retry > 0 {
				logger.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
			return "", logger.WarpError(err)
		}

		content, err := waitMessage(chatResponse, plugin.ToolCallCancel)
		if err != nil {
			if retry > 0 {
				logger.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
		}

		return content, logger.WarpError(err)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
