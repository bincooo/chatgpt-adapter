package coze

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	echo := ctx.GetBool(vars.GinEcho)

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
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

		if echo {
			bytes, _ := json.MarshalIndent(pMessages, "", "  ")
			logger.Infof("toolCall message: \n%s", bytes)
			return "", nil
		}

		co, msToken := extCookie(cookie)
		options, mode, err := newOptions(proxies, completion.Model, pMessages)
		if err != nil {
			return "", logger.WarpError(err)
		}

		chat := coze.New(co, msToken, options)
		chat.Session(plugin.HTTPClient)
		var lock *common.ExpireLock
		if mode == 'o' {
			l, e := draftBot(ctx, pMessages[0], chat, completion)
			if e != nil {
				return "", logger.WarpError(e.Err)
			}
			lock = l
		}

		query := ""
		if mode == 'w' {
			query = pMessages[len(pMessages)-1].Content
			chat.WebSdk(chat.TransferMessages(pMessages[:len(pMessages)-1]))
		} else {
			query = coze.MergeMessages(pMessages)
		}

		chatResponse, err := chat.Reply(common.GetGinContext(ctx), coze.Text, query)
		// 构建完请求即可解锁
		if lock != nil {
			lock.Unlock()
			botId := customBotId(completion.Model)
			rmLock(botId)
			logger.Infof("构建完成解锁：%s", botId)
		}

		if err != nil {
			return "", logger.WarpError(err)
		}

		return waitMessage(chatResponse, plugin.ToolCallCancel)
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
