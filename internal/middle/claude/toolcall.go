package claude

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	api "github.com/bincooo/claude-api"
	"github.com/bincooo/claude-api/types"
	"github.com/bincooo/claude-api/vars"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
)

func completeToolCalls(ctx *gin.Context, cookie, proxies string, completion pkg.ChatCompletion) bool {
	logrus.Infof("completeTools ...")
	exec, err := middle.CompleteToolCalls(ctx, completion, func(message string) (string, error) {
		model := vars.Model4WebClaude2
		if strings.HasPrefix(completion.Model, "claude-") {
			model = completion.Model
		}

		options := api.NewDefaultOptions(cookie, model)
		options.Proxies = proxies

		chat, err := api.New(options)
		if err != nil {
			return "", err
		}

		message = common.PadText(padMaxCount-len(message), message)
		chatResponse, err := chat.Reply(ctx.Request.Context(), "", []types.Attachment{
			{
				Content:  message,
				FileName: "paste.txt",
				FileSize: len(message),
				FileType: "text/plain",
			},
		})
		if err != nil {
			return "", err
		}

		defer chat.Delete()
		return waitMessage(chatResponse, middle.ToolCallCancel)
	})

	if err != nil {
		middle.ErrResponse(ctx, -1, err)
		return true
	}

	return exec
}
