package zed

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
	Model = "zed"
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if len(model) <= 4 || Model+"/" != model[:4] {
		return
	}
	slice := api.env.GetStringSlice("zed.model")
	for _, mod := range append(slice, []string{
		"claude-3-5-sonnet-latest",
		"claude-3-7-sonnet-latest",
	}...) {
		if model[4:] == mod {
			ok = true
			return
		}
	}
	return
}

func (api *api) Models() (slice []model.Model) {
	for _, mod := range append(api.env.GetStringSlice("zed.model"), []string{
		"claude-3-5-sonnet-latest",
		"claude-3-7-sonnet-latest",
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

	request, err := convertRequest(ctx, completion)
	if err != nil {
		logger.Error(err)
		return
	}

	r, err := fetch(ctx, api.env, proxied, cookie, request)
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
