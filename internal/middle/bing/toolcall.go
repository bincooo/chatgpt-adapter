package bing

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logrus.Infof("completeTools ...")
	exec, err := middle.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		options, err := edge.NewDefaultOptions(cookie, "")
		if err != nil {
			return "", err
		}

		chat := edge.New(options.
			Proxies(proxies).
			TopicToE(true).
			Notebook(true).
			Model(edge.ModelSydney))
		chatResponse, err := chat.Reply(ctx.Request.Context(), message, nil)
		if err != nil {
			return "", err
		}

		return waitMessage(chatResponse, middle.ToolCallCancel)
	})

	if err != nil {
		middle.ErrResponse(ctx, -1, err)
		return true
	}

	return exec
}
