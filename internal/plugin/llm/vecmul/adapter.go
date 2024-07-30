package vecmul

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	vec "github.com/bincooo/vecmul.com"
	"github.com/gin-gonic/gin"
)

var (
	Adapter = API{}
	Model   = "vecmul"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case Model + "/" + vec.GPT35,
		Model + "/" + vec.GPT4,
		Model + "/" + vec.GPT4o,
		Model + "/" + vec.Claude3Sonnet,
		Model + "/" + vec.Claude35Sonnet,
		Model + "/" + vec.Claude3Opus,
		Model + "/" + vec.Gemini15flash,
		Model + "/" + vec.Gemini15pro:
		return true
	default:
		return false
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.GPT35,
		},
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.GPT4,
		},
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.GPT4o,
		},
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.Claude3Sonnet,
		},
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.Claude35Sonnet,
		},
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.Claude3Opus,
		},
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.Gemini15flash,
		},
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "/" + vec.Gemini15pro,
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		proxies = ctx.GetString("proxies")

		echo       = ctx.GetBool(vars.GinEcho)
		token      = ctx.GetString("token")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, proxies, completion) {
			return
		}
	}

	chat := vec.New(proxies, completion.Model[7:], token)
	chat.Session(plugin.HTTPClient)
	message, tokens, err := mergeMessages(ctx, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if echo {
		response.Echo(ctx, completion.Model, message, completion.Stream)
		return
	}

	// 清理多余的标签
	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	ctx.Set(ginTokens, tokens)

	data, err := chat.Reply(common.GetGinContext(ctx), message, "")
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	content := waitResponse(ctx, matchers, cancel, data, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func joinMatchers(ctx *gin.Context, matchers []common.Matcher) (chan error, []common.Matcher) {
	// 自定义标记块中断
	cancel, matcher := common.NewCancelMatcher(ctx)
	matchers = append(matchers, matcher...)
	return cancel, matchers
}
