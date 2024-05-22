package cohere

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/plugin"
	coh "github.com/bincooo/cohere-api"
	"github.com/gin-gonic/gin"
)

var (
	Adapter = API{}
	Model   = "cohere"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case coh.COMMAND,
		coh.COMMAND_R,
		coh.COMMAND_LIGHT,
		coh.COMMAND_LIGHT_NIGHTLY,
		coh.COMMAND_NIGHTLY,
		coh.COMMAND_R_PLUS:
		return true
	default:
		return false
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "command",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-r",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-light",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-light-nightly",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-nightly",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-r-plus",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		notebook   = ctx.GetBool("notebook")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	var system string
	var message string
	var pMessages []coh.Message
	var chat coh.Chat
	if notebook {
		message = mergeMessages(completion.Messages)
		ctx.Set(ginTokens, common.CalcTokens(message))
		chat = coh.New(cookie, completion.Temperature, completion.Model, false)
		chat.Proxies(proxies)
		chat.TopK(completion.TopK)
		chat.MaxTokens(completion.MaxTokens)
		chat.StopSequences([]string{
			"user:",
			"assistant:",
			"system:",
		})
	} else {
		var tokens = 0
		pMessages, system, message, tokens = mergeChatMessages(completion.Messages)
		ctx.Set(ginTokens, tokens)
		chat = coh.New(cookie, completion.Temperature, completion.Model, true)
		chat.Proxies(proxies)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), pMessages, system, message)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	waitResponse(ctx, matchers, chatResponse, completion.Stream)
}
