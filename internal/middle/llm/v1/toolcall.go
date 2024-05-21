package v1

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func completeToolCalls(ctx *gin.Context, req pkg.ChatCompletion) (bool, error) {
	logrus.Infof("completeTools ...")
	return middle.CompleteToolCalls(ctx, req, func(message string) (string, error) {
		response, err := fetchGpt35(ctx, req)
		if err != nil {
			return "", err
		}

		return waitMessage(response, middle.ToolCallCancel)
	})
}
