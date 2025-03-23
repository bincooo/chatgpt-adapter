package cursor

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"net/url"
	"strings"
)

var (
	Model = "cursor"
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if len(model) <= 7 || Model+"/" != model[:7] {
		return
	}
	slice := api.env.GetStringSlice("cursor.model")
	for _, mod := range append(slice, []string{
		"claude-3.5-sonnet",
		"gpt-4",
		"gpt-4o",
		"claude-3-opus",
		"cursor-fast",
		"cursor-small",
		"gpt-3.5-turbo",
		"gpt-4-turbo-2024-04-09",
		"gpt-4o-128k",
		"gemini-1.5-flash-500k",
		"claude-3-haiku-200k",
		"claude-3-5-sonnet-200k",
		"claude-3-5-sonnet-20241022",
		"claude-3.7-sonnet",
		"claude-3.7-sonnet-max",
		"claude-3.7-sonnet-thinking",
		"claude-3.7-sonnet-thinking-max",
		"gpt-4o-mini",
		"o1-mini",
		"o1-preview",
		"o1",
		"claude-3.5-haiku",
		"gemini-exp-1206",
		"gemini-2.0-flash-thinking-exp",
		"gemini-2.0-flash-exp",
		"deepseek-v3",
		"deepseek-r1",
		"o3-mini",
	}...) {
		if model[7:] == mod {
			ok = true
			return
		}
	}
	return
}

func (api *api) Models() (slice []model.Model) {
	for _, mod := range append(api.env.GetStringSlice("cursor.model"), []string{
		"claude-3.5-sonnet",
		"gpt-4",
		"gpt-4o",
		"claude-3-opus",
		"cursor-fast",
		"cursor-small",
		"gpt-3.5-turbo",
		"gpt-4-turbo-2024-04-09",
		"gpt-4o-128k",
		"gemini-1.5-flash-500k",
		"claude-3-haiku-200k",
		"claude-3-5-sonnet-200k",
		"claude-3-5-sonnet-20241022",
		"claude-3.7-sonnet",
		"claude-3.7-sonnet-max",
		"claude-3.7-sonnet-thinking",
		"claude-3.7-sonnet-thinking-max",
		"gpt-4o-mini",
		"o1-mini",
		"o1-preview",
		"o1",
		"claude-3.5-haiku",
		"gemini-exp-1206",
		"gemini-2.0-flash-thinking-exp",
		"gemini-2.0-flash-exp",
		"deepseek-v3",
		"deepseek-r1",
		"o3-mini",
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

	cookie, err = url.QueryUnescape(cookie)
	if err != nil {
		return
	}

	if strings.Contains(cookie, "::") {
		cookie = strings.Split(cookie, "::")[1]
	}

	buffer, err := convertRequest(completion)
	if err != nil {
		return
	}

	r, err := fetch(ctx, api.env, cookie, buffer)
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
