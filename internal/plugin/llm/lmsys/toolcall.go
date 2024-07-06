package lmsys

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
)

func completeToolCalls(ctx *gin.Context, proxies, token string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	echo := ctx.GetBool(vars.GinEcho)

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		ch, err := fetch(common.GetGinContext(ctx), proxies, token, message, options{
			model:       completion.Model,
			temperature: completion.Temperature,
			topP:        completion.TopP,
			maxTokens:   completion.MaxTokens,
		})
		if err != nil {
			return "", err
		}

		return waitMessage(ch, plugin.ToolCallCancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
