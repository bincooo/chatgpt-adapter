package windsurf

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"strings"
)

var (
	Model = "windsurf"
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if len(model) <= 9 || Model+"/" != model[:9] {
		return
	}
	for mod := range mapModel {
		if model[9:] == mod {
			if strings.HasPrefix(mod, "deepseek") {
				completion := common.GetGinCompletion(ctx)
				completion.StopSequences = append(completion.StopSequences, "<codebase_search>", "<write_to_file>", "<open_link>")
				ctx.Set(vars.GinCompletion, completion)
			}
			ok = true
			return
		}
	}
	return
}

func (*api) Models() (slice []model.Model) {
	for mod := range mapModel {
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
		completion = common.GetGinCompletion(ctx)
	)

	if toolChoice(ctx, api.env, cookie, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		cookie     = ctx.GetString("token")
		completion = common.GetGinCompletion(ctx)
	)

	token, err := genToken(ctx.Request.Context(), api.env.GetString("server.proxied"), cookie)
	if err != nil {
		return
	}

	buffer, err := convertRequest(completion, cookie, token)
	if err != nil {
		return
	}

	r, err := fetch(ctx.Request.Context(), api.env, buffer)
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
