package bing

import (
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"time"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Infof("completeTools ...")
	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		retry := 3
		options, err := edge.NewDefaultOptions(cookie, "")
		if err != nil {
			return "", err
		}

	label:
		retry--
		chat := edge.New(options.
			Proxies(proxies).
			TopicToE(true).
			Model(edge.ModelSydney).
			Temperature(0.9))
		chatResponse, err := chat.Reply(ctx.Request.Context(), message, nil)
		if err != nil {
			if retry > 0 {
				logger.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
			return "", err
		}

		content, err := waitMessage(chatResponse, plugin.ToolCallCancel)
		if err != nil {
			if retry > 0 {
				logger.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
		}

		return content, err
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
