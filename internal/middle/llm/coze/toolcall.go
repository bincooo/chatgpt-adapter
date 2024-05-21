package coze

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logrus.Infof("completeTools ...")
	exec, err := middle.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		var notebook = ctx.GetBool("notebook")
		pMessages := []coze.Message{
			{
				Role:    "system",
				Content: message,
			},
		}

		co, msToken := extCookie(cookie)
		options := newOptions(proxies, pMessages)
		chat := coze.New(co, msToken, options)

		query := ""
		if notebook && len(pMessages) > 0 {
			// notebook 模式只取第一条 content
			query = pMessages[0].Content
		} else {
			query = coze.MergeMessages(pMessages)
		}

		chatResponse, err := chat.Reply(ctx.Request.Context(), query)
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
