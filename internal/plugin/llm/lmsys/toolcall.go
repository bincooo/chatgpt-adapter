package lmsys

import (
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
)

func completeToolCalls(ctx *gin.Context, proxies, token string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
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
