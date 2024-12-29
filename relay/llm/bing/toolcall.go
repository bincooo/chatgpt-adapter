package bing

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"context"
	"errors"
	"github.com/bincooo/edge-api"
	"github.com/bincooo/emit.io"
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

		newTok := false
	refresh:
		timeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		accessToken, err := genToken(timeout, cookie, newTok)
		if err != nil {
			return "", err
		}

		timeout, cancel = context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		conversationId, err := edge.CreateConversation(common.HTTPClient, timeout, accessToken)
		if err != nil {
			var hErr emit.Error
			if errors.As(err, &hErr) && hErr.Code == 401 && !newTok {
				newTok = true
				goto refresh
			}
			return "", err
		}

		timeout, cancel = context.WithTimeout(context.TODO(), 10*time.Second)
		defer cancel()
		defer edge.DeleteConversation(common.HTTPClient, timeout, conversationId, accessToken)

		challenge := ""
	label:
		buffer, err := edge.Chat(common.HTTPClient, ctx.Request.Context(), accessToken, conversationId, challenge, "", message)
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

		return waitMessage(buffer, toolcall.Cancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
