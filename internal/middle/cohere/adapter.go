package coh

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/cohere-api"
	"github.com/gin-gonic/gin"
)

var (
	Adapter = API{}
	Model   = "cohere"
)

type API struct {
	middle.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case cohere.COMMAND,
		cohere.COMMAND_R,
		cohere.COMMAND_LIGHT,
		cohere.COMMAND_LIGHT_NIGHTLY,
		cohere.COMMAND_NIGHTLY,
		cohere.COMMAND_R_PLUS:
		return true
	default:
		return false
	}
}

func (API) Models() []middle.Model {
	return []middle.Model{
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

	if common.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	var system string
	var message string
	var pMessages []cohere.Message
	var chat cohere.Chat
	if notebook {
		message = mergeMessages(completion.Messages)
		ctx.Set("tokens", common.CalcTokens(message))
		chat = cohere.New(cookie, completion.Temperature, completion.Model, false)
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
		ctx.Set("tokens", tokens)
		chat = cohere.New(cookie, completion.Temperature, completion.Model, true)
		chat.Proxies(proxies)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), pMessages, system, message)
	if err != nil {
		middle.ErrResponse(ctx, -1, err)
		return
	}

	waitResponse(ctx, matchers, chatResponse, completion.Stream)
}
