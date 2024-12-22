package windsurf

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

var (
	Model = "windsurf"
)

type api struct {
	inter.BaseAdapter

	env    *env.Environment
	holder *response.ContentHolder
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if len(model) <= 9 || Model+"/" != model[:9] {
		return
	}
	for _, mod := range []string{
		"claude-3-5-sonnet",
		"gpt4o",
	} {
		if model[9:] == mod {
			ok = true
			return
		}
	}
	return
}

func (*api) Models() (slice []model.Model) {
	for _, mod := range []string{
		"claude-3-5-sonnet",
		"gpt4o",
	} {
		slice = append(slice, model.Model{
			Id:      Model + "/" + mod,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		})
	}
	return
}

func (api *api) HandleMessages(ctx *gin.Context, completion model.Completion) (messages []model.Keyv[interface{}], err error) {
	var (
		toolMessages = toolcall.ExtractToolMessages(&completion)
	)

	if messages, err = api.holder.Handle(ctx, completion); err != nil {
		return
	}
	messages = append(messages, toolMessages...)
	return
}

func (api *api) ToolChoice(ctx *gin.Context) (ok bool, err error) {
	var (
		cookie     = ctx.GetString("token")
		completion = common.GetGinCompletion(ctx)
		echo       = ctx.GetBool(vars.GinEcho)
	)

	if echo {
		echoMessages(ctx, completion)
		return
	}

	if toolChoice(ctx, api.env, cookie, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		cookie     = ctx.GetString("token")
		completion = common.GetGinCompletion(ctx)
		echo       = ctx.GetBool(vars.GinEcho)
	)

	if echo {
		echoMessages(ctx, completion)
		return
	}

	token, err := genToken(ctx, api.env.GetString("server.proxied"), cookie)
	if err != nil {
		return
	}

	buffer, err := convertRequest(completion, cookie, token)
	if err != nil {
		return
	}

	r, err := fetch(ctx, api.env, buffer)
	if err != nil {
		logger.Error(err)
		return
	}

	content := waitResponse(ctx, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
	return
}
