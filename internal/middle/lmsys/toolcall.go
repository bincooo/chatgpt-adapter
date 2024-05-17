package lmsys

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

func completeToolCalls(ctx *gin.Context, proxies string, completion pkg.ChatCompletion) bool {
	logrus.Infof("completeTools ...")
	exec, err := middle.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		ch, err := fetch(ctx, proxies, message, options{
			model:       completion.Model,
			temperature: completion.Temperature,
			topP:        completion.TopP,
			maxTokens:   completion.MaxTokens,
		})
		if err != nil {
			return "", err
		}

		return waitMessage(ch, middle.ToolCallCancel)
	})

	if err != nil {
		errMessage := err.Error()
		if strings.Contains(errMessage, "Login verification is invalid") {
			middle.ErrResponse(ctx, http.StatusUnauthorized, errMessage)
		}
		middle.ErrResponse(ctx, -1, errMessage)
		return true
	}

	return exec
}
