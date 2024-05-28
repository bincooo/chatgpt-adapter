package coze

import (
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		var notebook = ctx.GetBool("notebook")
		pMessages := []coze.Message{
			{
				Role:    "system",
				Content: message,
			},
		}

		co, msToken := extCookie(cookie)
		options := newOptions(proxies, "", pMessages)
		chat := coze.New(co, msToken, options)

		query := ""
		if notebook && len(pMessages) > 0 {
			// notebook 模式只取第一条 content
			query = pMessages[0].Content
		} else {
			query = coze.MergeMessages(pMessages)
		}

		chatResponse, err := chat.Reply(ctx.Request.Context(), coze.Text, query)
		if err != nil {
			return "", err
		}

		return waitMessage(chatResponse, plugin.ToolCallCancel)
	})

	if err != nil {
		errMessage := err.Error()
		if strings.Contains(errMessage, "Login verification is invalid") {
			logger.Error(err)
			response.Error(ctx, http.StatusUnauthorized, errMessage)
		}
		logger.Error(err)
		response.Error(ctx, -1, errMessage)
		return true
	}

	return exec
}
