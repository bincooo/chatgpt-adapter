package you

import (
	"errors"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
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
	)

	defer func(cookies []string) {
		if len(cookies) == 0 {
			return
		}
		for _, value := range cookies {
			resetMarker(value)
		}
	}(cookies)

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
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
		chatResponse, err := chat.Reply(common.GetGinContext(ctx), []you.Message{
			{"", message},
		}, "Please review the attached prompt", true)
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
