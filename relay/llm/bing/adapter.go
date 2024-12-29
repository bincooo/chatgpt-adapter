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
	"errors"
	"github.com/bincooo/edge-api"
	"github.com/bincooo/emit.io"
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

	content, query := convertRequest(ctx, completion)
	newTok := false
refresh:
	timeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	accessToken, err := genToken(timeout, cookie, newTok)
	if err != nil {
		return
	}

	timeout, cancel = context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	conversationId, err := edge.CreateConversation(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), timeout, accessToken)
	if err != nil {
		var hErr emit.Error
		if errors.As(err, &hErr) && hErr.Code == 401 && !newTok {
			newTok = true
			goto refresh
		}
		return
	}

	timeout, cancel = context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	defer edge.DeleteConversation(common.HTTPClient, timeout, conversationId, accessToken)

	challenge := ""
label:
	message, err := edge.Chat(common.HTTPClient, ctx.Request.Context(), accessToken, conversationId, challenge, content,
		elseOf(query == "", "读取内容并以[\n\nAi:]角色继续回复", query))
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

	content = waitResponse(ctx, message, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
	return
}

func convertRequest(ctx *gin.Context, completion model.Completion) (content, query string) {
	countMax := 10240
	count := 0
	pos := 0
	for i := len(completion.Messages) - 1; i >= 0; i-- {
		if completion.Messages[i].Has("content") {
			count += len(completion.Messages[i].GetString("content"))
			if count > countMax {
				break
			}
		}
		pos = i
	}

	content = strings.Join(stream.Map(stream.OfSlice(completion.Messages[:pos]), func(message model.Keyv[interface{}]) string {
		convertRole, trun := response.ConvertRole(ctx, message.GetString("role"))
		return convertRole + message.GetString("content") + trun
	}).ToSlice(), "\n\n")

	query = strings.Join(stream.Map(stream.OfSlice(completion.Messages[pos:]), func(message model.Keyv[interface{}]) string {
		convertRole, trun := response.ConvertRole(ctx, message.GetString("role"))
		return convertRole + message.GetString("content") + trun
	}).ToSlice(), "\n\n")

	if query != "" {
		convertRole, _ := response.ConvertRole(ctx, "assistant")
		query += "\n\n" + convertRole
	}
	return
}

func genToken(ctx context.Context, ident string, new bool) (accessToken string, err error) {
	cacheManager := cache.BingCacheManager()
	accessToken, err = cacheManager.GetValue(ident)
	if !new && (err != nil || accessToken != "") {
		if accessToken != "" {
			accessToken = strings.Split(accessToken, "|")[1]
		}
		return
	}

	var nIdent = ident
	if accessToken != "" {
		split := strings.Split(ident, "|")
		if len(split) >= 3 {
			nIdent = strings.Join([]string{split[0], split[1], strings.Split(accessToken, "|")[0]}, "|")
		}
	}

	accessToken, err = edge.RefreshToken(common.HTTPClient, ctx, nIdent)
	if err != nil {
		return
	}

	err = cacheManager.SetWithExpiration(ident, accessToken, 48*time.Hour)
	accessToken = strings.Split(accessToken, "|")[1]
	return
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}
