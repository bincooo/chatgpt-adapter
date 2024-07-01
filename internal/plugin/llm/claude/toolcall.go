package claude

import (
	"errors"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	claude3 "github.com/bincooo/claude-api"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie string, completion pkg.ChatCompletion) bool {
	logger.Infof("completeTools ...")
	var (
		model = ""
		echo  = ctx.GetBool(vars.GinEcho)
	)

	exec, err := plugin.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		if echo {
			logger.Infof("toolCall message: \n%s", message)
			return "", nil
		}

		if strings.HasPrefix(completion.Model, "claude-") {
			if completion.Model != "claude-3" {
				model = completion.Model
			}
		}

		options, err := claude3.NewDefaultOptions(cookie, model)
		if err != nil {
			return "", logger.WarpError(err)
		}

		chat, err := claude3.New(options)
		if err != nil {
			return "", logger.WarpError(err)
		}

		if ctx.GetBool("pad") {
			count := ctx.GetInt("claude.pad")
			if count == 0 {
				count = padMaxCount
			}
			message = common.PadJunkMessage(count-len(message), message)
		}

		chatResponse, err := chat.Reply(common.GetGinContext(ctx), "", []claude3.Attachment{
			{
				Content:  message,
				FileName: "paste.txt",
				FileSize: len(message),
				FileType: "text/plain",
			},
		})
		if err != nil {
			var se emit.Error
			if errors.As(err, &se) {
				if se.Code == 429 {
					_ = claudeRollContainer.SetMarker(cookie, 2)
				}
			}
			return "", logger.WarpError(err)
		}

		defer chat.Delete()
		return waitMessage(chatResponse, plugin.ToolCallCancel)
	})

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return true
	}

	return exec
}
