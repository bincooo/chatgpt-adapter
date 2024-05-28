package claude

import (
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	claude2 "github.com/bincooo/claude-api"
	"github.com/bincooo/claude-api/vars"
	"github.com/gin-gonic/gin"
	"strings"
)

var (
	Adapter     = API{}
	Model       = "claude"
	padMaxCount = 25000
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case "claude",
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229":
		return true
	default:
		return false
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "claude",
			Object:  "model",
			Created: 1686935002,
			By:      "claude-adapter",
		}, {
			Id:      "claude-3-haiku-20240307",
			Object:  "model",
			Created: 1686935002,
			By:      "claude-adapter",
		}, {
			Id:      "claude-3-sonnet-20240229",
			Object:  "model",
			Created: 1686935002,
			By:      "claude-adapter",
		}, {
			Id:      "claude-3-opus-20240229",
			Object:  "model",
			Created: 1686935002,
			By:      "claude-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")

		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
		model      = vars.Model4WebClaude2
	)

	if strings.HasPrefix(completion.Model, "claude-") {
		model = completion.Model
	}

	options := claude2.NewDefaultOptions(cookie, model)
	options.Proxies = proxies

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	attachments, tokens := mergeMessages(completion.Messages)
	ctx.Set(ginTokens, tokens)
	chat, err := claude2.New(options)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), "", attachments)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	defer chat.Delete()
	content := waitResponse(ctx, matchers, chatResponse, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}
