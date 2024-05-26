package cohere

import (
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/cohere-api"
	"github.com/gin-gonic/gin"
	"net/http"
	"regexp"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		chat := cohere.New(cookie, 0.4, completion.Model, false)
		chat.Proxies(proxies)
		chat.TopK(completion.TopK)
		chat.MaxTokens(completion.MaxTokens)
		chat.StopSequences([]string{
			"user:",
			"assistant:",
			"system:",
		})

		message = regexp.MustCompile("工具推荐： toolId = .{5}").
			ReplaceAllString(message, "")
		chatResponse, err := chat.Reply(ctx.Request.Context(), make([]cohere.Message, 0), "", message, cohere.ToolObject{})
		if err != nil {
			return "", err
		}

		return waitMessage(chatResponse, plugin.ToolCallCancel)
	})

	if err != nil {
		errMessage := err.Error()
		if strings.Contains(errMessage, "Login verification is invalid") {
			response.Error(ctx, http.StatusUnauthorized, errMessage)
		} else {
			response.Error(ctx, -1, errMessage)
		}
		logger.Error(err)
		return true
	}

	return exec
}
