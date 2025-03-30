package qodo

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
	Model = "qodo"
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if len(model) <= 5 {
		return
	}

	slice := api.env.GetStringSlice("qodo.model")
	for _, mod := range append(slice, []string{
		"claude-3-5-sonnet",
		"claude-3-7-sonnet",
		"gpt-4o",
		"o1",
		"o3-mini",
		"o3-mini-high",
		"gemini-2.0-flash",
		"deepseek-r1",
		"deepseek-r1-full",
	}...) {
		if model[5:] == mod {
			ok = true
			return
		}
	}
	return
}

func (api *api) Models() (slice []model.Model) {
	for _, mod := range append(api.env.GetStringSlice("qodo.model"), []string{
		"claude-3-5-sonnet",
		"claude-3-7-sonnet",
		"gpt-4o",
		"o1",
		"o3-mini",
		"o3-mini-high",
		"gemini-2.0-flash",
		"deepseek-r1",
		"deepseek-r1-full",
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
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	request, err := convertRequest(ctx, api.env, completion)
	if err != nil {
		logger.Error(err)
		return
	}

	r, err := fetch(ctx, proxied, request)
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
