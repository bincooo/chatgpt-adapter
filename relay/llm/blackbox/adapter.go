package blackbox

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

var (
	Model = "blackbox"
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if len(model) <= 9 || Model+"/" != model[:9] {
		return
	}

	slice := api.env.GetStringSlice("blackbox.model")
	for _, mod := range append(slice, []string{
		"GPT-4o",
		"Gemini-PRO",
		"Claude-Sonnet-3.5",
		"Claude-Sonnet-3.7",
		"DeepSeek-V3",
		"DeepSeek-R1",
	}...) {
		if model[9:] == mod {
			ok = true
			return
		}
	}
	return
}

func (api *api) Models() (slice []model.Model) {
	s := api.env.GetStringSlice("blackbox.model")
	for _, mod := range append(s, []string{
		"GPT-4o",
		"Gemini-PRO",
		"Claude-Sonnet-3.5",
		"Claude-Sonnet-3.7",
		"DeepSeek-V3",
		"DeepSeek-R1",
	}...) {
		slice = append(slice, model.Model{
			Id:      Model + "/" + mod,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		})
	}
	return
}

func (api *api) ToolChoice(ctx *gin.Context) (ok bool, err error) {
	var (
		cookie     = ctx.GetString("token")
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	if toolChoice(ctx, api.env, cookie, proxied, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		cookie     = ctx.GetString("token")
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	request := convertRequest(ctx, api.env, completion)
	r, err := fetch(ctx.Request.Context(), proxied, cookie, request)
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
