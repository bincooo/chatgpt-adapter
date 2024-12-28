package windsurf

import (
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

func toolChoice(ctx *gin.Context, env *env.Environment, cookie string, completion model.Completion) bool {
	logger.Info("completeTools ...")
	echo := ctx.GetBool(vars.GinEcho)

	exec, err := toolcall.ToolChoice(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		completion.Messages = []model.Keyv[interface{}]{
			{
				"role":    "user",
				"content": message,
			},
		}

		token, err := genToken(ctx.Request.Context(), env.GetString("server.proxied"), cookie)
		if err != nil {
			return "", err
		}

		messageBuffer, err := convertRequest(completion, cookie, token)
		if err != nil {
			return "", err
		}

		r, err := fetch(ctx.Request.Context(), env, messageBuffer)
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
