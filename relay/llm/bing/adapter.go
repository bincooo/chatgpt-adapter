package bing

import (
	"chatgpt-adapter/core/cache"
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
		proxied    = api.env.GetBool("bing.proxied")
	)

	if echo {
		echoMessages(ctx, completion)
		return
	}

	query := ""
	if i := len(completion.Messages) - 1; completion.Messages[i].Is("role", "user") {
		query = completion.Messages[i].GetString("content")
		completion.Messages = completion.Messages[:i]
	}
	request := convertRequest(ctx, completion)

	timeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	accessToken, err := genToken(timeout, cookie)
	if err != nil {
		return
	}

	timeout, cancel = context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	conversationId, err := edge.CreateConversation(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), timeout, accessToken)
	if err != nil {
		return
	}

	timeout, cancel = context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	defer edge.DeleteConversation(common.HTTPClient, timeout, conversationId, accessToken)

	challenge := ""
label:
	message, err := edge.Chat(common.HTTPClient, ctx.Request.Context(), accessToken, conversationId, challenge, request, "从[\n\nAi:]处继续回复，\n\n当前问题是: "+query)
	if err != nil {
		if challenge == "" && err.Error() == "challenge" {
			challenge, err = hookCloudflare()
			if err != nil {
				return
			}
			goto label
		}
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

func genToken(ctx context.Context, ident string) (accessToken string, err error) {
	cacheManager := cache.BingCacheManager()
	accessToken, err = cacheManager.GetValue(ident)
	if err != nil || accessToken != "" {
		return
	}

	accessToken, err = edge.RefreshToken(common.HTTPClient, ctx, ident)
	if err != nil {
		return
	}

	err = cacheManager.SetWithExpiration(ident, accessToken, 12*time.Hour)
	return
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}
