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
	"github.com/iocgo/sdk/env"
	"time"
)

func toolChoice(ctx *gin.Context, completion model.Completion) bool {
	logger.Info("completeTools ...")
	echo := ctx.GetBool(vars.GinEcho)
	cookie, _ := common.GetGinValue[map[string]string](ctx, "token")
	proxied := env.Env.GetBool("bing.proxied")

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
		accessToken, err := genToken(timeout, cookie, proxied, newTok)
		if err != nil {
			return "", err
		}

		timeout, cancel = context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		conversationId, err := edge.CreateConversation(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), timeout, accessToken)
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
		defer edge.DeleteConversation(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), timeout, conversationId, accessToken)

		challenge := ""
	label:
		buffer, err := edge.Chat(elseOf(proxied, common.HTTPClient, common.NopHTTPClient),
			ctx.Request.Context(),
			accessToken,
			conversationId,
			challenge, "", message, "",
			elseOf[byte](completion.Model == Model, 0, 1))
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
