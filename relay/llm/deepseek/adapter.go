package deepseek

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
	Model = "deepseek"
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if len(model) <= 9 || Model+"-" != model[:9] {
		return
	}

	ok = model[9:] == "chat" || model[9:] == "reasoner"
	return
}

func (api *api) Models() (slice []model.Model) {
	slice = append(slice, model.Model{
		Id:      Model + "-chat",
		Object:  "model",
		Created: 1686935002,
		By:      Model + "-adapter",
	})
	slice = append(slice, model.Model{
		Id:      Model + "-reasoner",
		Object:  "model",
		Created: 1686935002,
		By:      Model + "-adapter",
	})
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

	request, err := convertRequest(ctx, api.env, completion)
	if err != nil {
		logger.Error(err)
		return
	}

	r, err := fetch(ctx.Request.Context(), proxied, cookie, request)
	if err != nil {
		logger.Error(err)
		return
	}

	defer deleteSession(ctx, api.env, request.ChatSessionId)
	content := waitResponse(ctx, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
	return
}
