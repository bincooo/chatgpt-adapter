package you

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
)

func toolChoice(ctx *gin.Context, cookie, proxies string, completion model.Completion) bool {
	logger.Infof("completeTools ...")

	var (
		echo = ctx.GetBool(vars.GinEcho)
	)

	exec, err := toolcall.ToolChoice(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		chat := you.New(cookie, completion.Model, proxies)
		chat.LimitWithE(true)
		chat.Client(common.HTTPClient)
		clearance := ctx.GetString("clearance")
		if clearance != "" {
			chat.CloudFlare(clearance, ctx.GetString("userAgent"), ctx.GetString("lang"))
		}

		chatResponse, err := chat.Reply(ctx.Request.Context(), nil, message, "Please review the attached prompt")
		if err != nil {
			return "", err
		}

		return waitMessage(chatResponse, toolcall.Cancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
