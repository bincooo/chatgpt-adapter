package coh

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/cohere-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logrus.Infof("completeTools ...")
	exec, err := middle.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		pMessages := make([]cohere.Message, 0)
		chat := cohere.New(cookie, 0.4, completion.Model, false)
		chat.Proxies(proxies)
		chat.TopK(completion.TopK)
		chat.MaxTokens(completion.MaxTokens)
		chat.StopSequences([]string{
			"user:",
			"assistant:",
			"system:",
		})

		chatResponse, err := chat.Reply(ctx.Request.Context(), pMessages, "", message)
		if err != nil {
			return "", err
		}

		return waitMessage(chatResponse, middle.ToolCallCancel)
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
