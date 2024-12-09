package bing

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"context"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/stream"
	"strings"
	"time"
)

var (
	Model = "bing"
)

type api struct {
	inter.BaseAdapter

	env    *env.Environment
	holder *response.ContentHolder
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	ok = Model == model
	return
}

func (*api) Models() (slice []model.Model) {
	slice = append(slice, model.Model{
		Id:      Model,
		Object:  "model",
		Created: 1686935002,
		By:      Model + "-adapter",
	})
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
		completion = common.GetGinCompletion(ctx)
		echo       = ctx.GetBool(vars.GinEcho)
	)

	if echo {
		echoMessages(ctx, completion)
		return
	}

	if toolChoice(ctx, completion) {
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

	request := convertRequest(ctx, completion)

	timeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	conversationId, err := edge.CreateConversation(common.HTTPClient, timeout, cookie)
	if err != nil {
		return
	}

	timeout, cancel = context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	defer edge.DeleteConversation(common.HTTPClient, timeout, conversationId, cookie)

	message, err := edge.Chat(common.HTTPClient, ctx.Request.Context(), cookie, conversationId, request)
	if err != nil {
		return
	}

	content := waitResponse(ctx, message, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
	return
}

func convertRequest(ctx *gin.Context, completion model.Completion) (content string) {
	content = strings.Join(stream.Map(stream.OfSlice(completion.Messages), func(message model.Keyv[interface{}]) string {
		convertRole, trun := response.ConvertRole(ctx, message.GetString("role"))
		return convertRole + message.GetString("content") + trun
	}).ToSlice(), "\n\n")
	if content != "" {
		convertRole, _ := response.ConvertRole(ctx, "assistant")
		content += "\n\n" + convertRole
	}
	return
}
