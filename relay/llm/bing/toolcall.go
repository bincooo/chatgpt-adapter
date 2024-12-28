package bing

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"context"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"time"
)

func toolChoice(ctx *gin.Context, completion model.Completion) bool {
	logger.Info("completeTools ...")
	echo := ctx.GetBool(vars.GinEcho)
	cookie := ctx.GetString("token")

	exec, err := toolcall.ToolChoice(ctx, completion, func(message string) (string, error) {
		message += "\n\nAi:"
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		timeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		conversationId, err := edge.CreateConversation(common.HTTPClient, timeout, cookie)
		if err != nil {
			return "", err
		}

		challenge := ""
	label:
		buffer, err := edge.Chat(common.HTTPClient, ctx.Request.Context(), cookie, conversationId, challenge, message, "从[\n\nAi:]处继续回复")
		if err != nil {
			if challenge == "" && err.Error() == "challenge" {
				challenge, err = hookCloudflare()
				if err != nil {
					return "", err
				}
				goto label
			}
			return "", err
		}

		return waitMessage(buffer, nil, toolcall.Cancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
