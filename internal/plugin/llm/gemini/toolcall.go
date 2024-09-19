package gemini

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Info("completeTools ...")
	echo := ctx.GetBool(vars.GinEcho)

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		message = strings.TrimSpace(message)
		var messages []map[string]interface{}
		messages = append(messages, map[string]interface{}{
			"role": "user",
			"parts": []interface{}{
				map[string]string{
					"text": message,
				},
			},
		})

		if echo {
			bytes, _ := json.MarshalIndent(messages, "", "  ")
			logger.Infof("toolCall message: \n%s", bytes)
			return "", nil
		}

		completion.Tools = nil
		completion.ToolChoice = nil
		r, err := build(common.GetGinContext(ctx), proxies, cookie, messages, completion)
		if err != nil {
			return "", err
		}

		return waitMessage(r, plugin.ToolCallCancel)
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
