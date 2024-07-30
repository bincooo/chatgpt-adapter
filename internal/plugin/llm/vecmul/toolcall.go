package vecmul

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"github.com/bincooo/vecmul.com"
	"github.com/gin-gonic/gin"
)

func completeToolCalls(ctx *gin.Context, proxies string, completion pkg.ChatCompletion) bool {
	logger.Infof("completeTools ...")

	var (
		echo  = ctx.GetBool(vars.GinEcho)
		token = ctx.GetString("token")
	)

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		chat := vecmul.New(proxies, completion.Model[7:], token)
		chat.Session(plugin.HTTPClient)
		data, err := chat.Reply(common.GetGinContext(ctx), message, "")
		if err != nil {
			return "", logger.WarpError(err)
		}

		content, err := waitMessage(data, plugin.ToolCallCancel)
		return content, logger.WarpError(err)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
