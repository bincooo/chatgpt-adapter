package lmsys

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
)

func completeToolCalls(ctx *gin.Context, proxies string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		ch, err := fetch(ctx.Request.Context(), proxies, message, options{
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
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
