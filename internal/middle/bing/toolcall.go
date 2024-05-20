package bing

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logrus.Infof("completeTools ...")
	exec, err := middle.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		retry := 3
		options, err := edge.NewDefaultOptions(cookie, "")
		if err != nil {
			return "", err
		}

		// message = strings.ReplaceAll(message, "<|system|>", "<|user|>")
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
				logrus.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
			return "", err
		}

		content, err := waitMessage(chatResponse, middle.ToolCallCancel)
		if err != nil {
			if retry > 0 {
				logrus.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
		}

		return content, err
	})

	if err != nil {
		middle.ErrResponse(ctx, -1, err)
		return true
	}

	return exec
}
