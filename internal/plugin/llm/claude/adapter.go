package claude

import (
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	claude3 "github.com/bincooo/claude-api"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

var (
	Adapter     = API{}
	Model       = "claude"
	padMaxCount = 25000

	claudeRollContainer *common.PollContainer[string]
)

type API struct {
	plugin.BaseAdapter
}

func init() {
	common.AddInitialized(func() {
		cookies := pkg.Config.GetStringSlice("claude.cookies")
		if len(cookies) == 0 {
			return
		}
		claudeRollContainer = common.NewPollContainer[string](cookies, 30*time.Minute)
		claudeRollContainer.Condition = Condition
	})
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case "claude",
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-5-sonnet-20240620",
		"claude-3-opus-20240229":
		return true
	default:
		return strings.HasPrefix(model, "claude-")
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "claude-3",
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
			Id:      "claude-3-5-sonnet-20240620",
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
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
		model      = ""
	)

	if strings.HasPrefix(completion.Model, "claude-") {
		if completion.Model != "claude-3" {
			model = completion.Model
		}
	}

	cookie, err := claudeRollContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	defer resetMarker(cookie)
	options, err := claude3.NewDefaultOptions(cookie, model)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, completion) {
			return
		}
	}

	attachments, tokens := mergeMessages(ctx, completion.Messages)
	ctx.Set(ginTokens, tokens)
	chat, err := claude3.New(options)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	chat.Client(plugin.HTTPClient)
	chatResponse, err := chat.Reply(common.GetGinContext(ctx), "", attachments)
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

func Condition(cookie string) bool {
	marker, err := claudeRollContainer.GetMarker(cookie)
	if err != nil {
		logger.Error(err)
		return false
	}

	return marker == 0
}

func resetMarker(cookie string) {
	marker, e := claudeRollContainer.GetMarker(cookie)
	if e != nil {
		logger.Error(e)
		return
	}

	if marker == 0 || marker > 1 {
		return
	}

	e = claudeRollContainer.SetMarker(cookie, 0)
	if e != nil {
		logger.Error(e)
	}
}
