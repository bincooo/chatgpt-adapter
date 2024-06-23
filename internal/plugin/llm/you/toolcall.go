package you

import (
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Infof("completeTools ...")
	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		retry := 3
	label:
		retry--
		chat := you.New(cookie, completion.Model, proxies)
		chat.LimitWithE(true)
		chat.Client(plugin.HTTPClient)

		if err := tryCloudFlare(ctx); err != nil {
			return "", logger.WarpError(err)
		}

		chat.CloudFlare(clearance, userAgent)
		chatResponse, err := chat.Reply(common.GetGinContext(ctx), nil, message, false)
		if err != nil {
			if retry > 0 {
				if strings.Contains(err.Error(), "ZERO QUOTA") {
					return "", err
				}

				logger.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
			return "", logger.WarpError(err)
		}

		content, err := waitMessage(chatResponse, plugin.ToolCallCancel)
		if err != nil {
			if retry > 0 {
				logger.Errorf("Failed to complete tool calls: %v", err)
				time.Sleep(time.Second)
				goto label
			}
		}

		return content, logger.WarpError(err)
	})

	if err != nil {
		logger.Error(err)
		// 交给下一步处理
		if strings.Contains(err.Error(), "ZERO QUOTA") {
			return false
		}

		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
