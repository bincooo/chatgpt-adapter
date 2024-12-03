package lmsys

import (
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

func toolChoice(ctx *gin.Context, env *env.Environment, proxies string, completion model.Completion) bool {
	logger.Info("completeTools ...")
	echo := ctx.GetBool(vars.GinEcho)

	exec, err := toolcall.ToolChoice(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		ch, err := fetch(ctx.Request.Context(), env, proxies, message,
			options{
				model:       completion.Model,
				temperature: completion.Temperature,
				topP:        completion.TopP,
				maxTokens:   completion.MaxTokens,
			})
		if err != nil {
			return "", err
		}

		return waitMessage(ch, toolcall.Cancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
