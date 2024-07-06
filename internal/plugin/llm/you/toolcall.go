package you

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"errors"
	"github.com/bincooo/emit.io"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logger.Infof("completeTools ...")

	var (
		code    = -1
		cookies []string

		echo = ctx.GetBool(vars.GinEcho)
	)

	defer func() {
		for _, value := range cookies {
			resetMarker(value)
		}
	}()

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		retry := 3
	label:
		retry--
		chat := you.New(cookie, completion.Model, proxies)
		chat.LimitWithE(true)
		chat.Client(plugin.HTTPClient)

		if err := tryCloudFlare(); err != nil {
			return "", logger.WarpError(err)
		}

		chat.CloudFlare(clearance, userAgent, lang)
		chatResponse, err := chat.Reply(common.GetGinContext(ctx), nil, message, "Please review the attached prompt")
		if err != nil {
			if strings.Contains(err.Error(), "ZERO QUOTA") {
				code = 429
				_ = youRollContainer.SetMarker(cookie, 2)
				co, e := youRollContainer.Poll()
				if e != nil {
					return "", logger.WarpError(e)
				}
				cookie = co
				cookies = append(cookies, cookie)
			}

			var se emit.Error
			if errors.As(err, &se) {
				if se.Code == 403 {
					cleanCf()
				}
			}

			if retry > 0 {
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
		response.Error(ctx, code, err)
		return true
	}

	if code == 429 {
		response.Error(ctx, code, err)
		return true
	}

	return exec
}
