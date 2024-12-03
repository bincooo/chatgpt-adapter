package v1

import (
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
)

func toolChoice(ctx *gin.Context, proxies string, completion model.Completion) bool {
	logger.Info("tool choice ...")
	cookie := ctx.GetString("token")
	exec, err := toolcall.ToolChoice(ctx, completion, func(message string) (string, error) {
		completion.Stream = true
		completion.Messages = []model.Keyv[interface{}]{
			{
				"role":    "user",
				"content": message,
			},
		}

		r, err := fetch(ctx, proxies, cookie, completion)
		if err != nil {
			return "", err
		}

		return waitMessage(r, toolcall.Cancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
