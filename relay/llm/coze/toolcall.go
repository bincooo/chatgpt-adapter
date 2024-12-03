package coze

import (
	"net/http"
	"strings"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
)

func toolChoice(ctx *gin.Context, cookie, proxies string, completion model.Completion) bool {
	logger.Info("completeTools ...")
	exec, err := toolcall.ToolChoice(ctx, completion, func(message string) (string, error) {
		message = strings.TrimSpace(message)
		system := ""
		if strings.HasPrefix(message, "<|system|>") {
			index := strings.Index(message, "<|end|>")
			system = message[:index+7]
			message = strings.TrimSpace(message[index+7:])
		}

		var pMessages []coze.Message
		if system != "" {
			pMessages = append(pMessages, coze.Message{
				Role:    "system",
				Content: system,
			})
		}

		pMessages = append(pMessages, coze.Message{
			Role:    "user",
			Content: message,
		})

		co, msToken := extCookie(cookie)
		options, mode, err := newOptions(proxies, completion.Model)
		if err != nil {
			return "", err
		}

		chat := coze.New(co, msToken, options)
		chat.Session(common.HTTPClient)

		query := ""
		if mode == 'w' {
			query = pMessages[len(pMessages)-1].Content
			chat.WebSdk(chat.TransferMessages(pMessages[:len(pMessages)-1]))
		} else {
			query = coze.MergeMessages(pMessages)
		}

		chatResponse, err := chat.Reply(ctx.Request.Context(), coze.Text, query)
		if err != nil {
			return "", err
		}

		return waitMessage(chatResponse, toolcall.Cancel)
	})

	if err != nil {
		errMessage := err.Error()
		if strings.Contains(errMessage, "Login verification is invalid") {
			logger.Error(err)
			response.Error(ctx, http.StatusUnauthorized, errMessage)
			return true
		}

		logger.Error(err)
		response.Error(ctx, -1, errMessage)
		return true
	}

	return exec
}
