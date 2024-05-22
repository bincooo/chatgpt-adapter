package v1

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
)

func completeToolCalls(ctx *gin.Context, req pkg.ChatCompletion) (bool, error) {
	logger.Info("completeTools ...")
	return plugin.CompleteToolCalls(ctx, req, func(message string) (string, error) {
		response, err := fetchGpt35(ctx, req)
		if err != nil {
			return "", err
		}

		return waitMessage(response, plugin.ToolCallCancel)
	})
}
