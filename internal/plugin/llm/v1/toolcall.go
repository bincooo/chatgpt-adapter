package v1

import (
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
)

func completeToolCalls(ctx *gin.Context, proxies string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	cookie := ctx.GetString("token")
	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		completion.Stream = true
		completion.Messages = []pkg.Keyv[interface{}]{
			{
				"role":    "user",
				"content": message,
			},
		}

		r, err := fetch(ctx, proxies, cookie, completion)
		if err != nil {
			return "", err
		}

		return waitMessage(r, plugin.ToolCallCancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
