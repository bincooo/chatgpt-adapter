package bing

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"sync"
	"time"

	"chatgpt-adapter/core/cache"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"github.com/bincooo/edge-api"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/stream"
)

var (
	Model = "bing"
	mu    sync.Mutex
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	var token = ctx.GetString("token")
	ok = Model == model || model == Model+"-reason"
	if ok {
		password := api.env.GetString("server.password")
		if password != "" && password != token {
			err = response.UnauthorizedError
			return
		}
	}
	return
}

func (*api) Models() (slice []model.Model) {
	slice = append(slice, model.Model{
		Id:      Model,
		Object:  "model",
		Created: 1686935002,
		By:      Model + "-adapter",
	})
	slice = append(slice, model.Model{
		Id:      Model + "-reason",
		Object:  "model",
		Created: 1686935002,
		By:      Model + "-adapter",
	})
	return
}

func (api *api) ToolChoice(ctx *gin.Context) (ok bool, err error) {
	var (
		completion = common.GetGinCompletion(ctx)
	)

	if toolChoice(ctx, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		cookie, _  = common.GetGinValue[map[string]string](ctx, "token")
		completion = common.GetGinCompletion(ctx)
		proxied    = api.env.GetBool("bing.proxied")
	)

	content, query, attr := convertRequest(ctx, completion)
	newTok := false
refresh:
	timeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	accessToken, err := genToken(timeout, cookie, proxied, newTok)
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
	defer edge.DeleteConversation(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), timeout, conversationId, accessToken)

	if attr != "" {
		attr, err = extAttr(ctx, proxied, attr, accessToken)
		if err != nil {
			return
		}
	}

	challenge := ""
label:
	message, err := edge.Chat(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), ctx.Request.Context(),
		accessToken,
		conversationId,
		challenge,
		content,
		elseOf(query == "", "读取内容并以[\n\nAi:]角色继续回复", query), attr, elseOf[byte](completion.Model == Model, 0, 1))
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

func extAttr(ctx *gin.Context, proxied bool, attr, accessToken string) (ret string, err error) {
	var buffer []byte
	if strings.HasPrefix(attr, "http") {
		buffer, err = common.DownloadBuffer(common.HTTPClient, "", attr, nil)
	} else if strings.HasPrefix(attr, "data:image/") {
		if pos := strings.Index(attr, ";"); pos > 0 {
			attr = attr[pos+1:]
		}
		if strings.HasPrefix(attr, "base64,") {
			attr = attr[7:]
		}
		buffer, err = base64.StdEncoding.DecodeString(attr)
	}
	if err != nil {
		return
	}

	ret, err = edge.Attachments(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), ctx.Request.Context(), buffer, accessToken)
	return
}

func convertRequest(ctx *gin.Context, completion model.Completion) (content, query, attr string) {
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

	multiMessage := func(convertRole string, message model.Keyv[interface{}]) string {
		values := message.GetSlice("content")
		text := ""
		for _, value := range values {
			var ok bool
			message, ok = value.(map[string]interface{})
			if !ok {
				continue
			}
			if message.Is("type", "text") {
				text = convertRole + message.GetString("text")
			}
			if message.Is("type", "image_url") {
				o := message.GetKeyv("image_url")
				attr = o.GetString("url")
			}
		}
		return text
	}

	content = strings.Join(stream.Map(stream.OfSlice(completion.Messages[:pos]), func(message model.Keyv[interface{}]) string {
		convertRole, trun := response.ConvertRole(ctx, message.GetString("role"))
		if !message.IsString("content") {
			return convertRole + multiMessage(convertRole, message) + trun
		}
		return convertRole + message.GetString("content") + trun
	}).ToSlice(), "\n\n")

	query = strings.Join(stream.Map(stream.OfSlice(completion.Messages[pos:]), func(message model.Keyv[interface{}]) string {
		convertRole, trun := response.ConvertRole(ctx, message.GetString("role"))
		if !message.IsString("content") {
			return convertRole + multiMessage(convertRole, message) + trun
		}
		return convertRole + message.GetString("content") + trun
	}).ToSlice(), "\n\n")

	if query != "" {
		convertRole, _ := response.ConvertRole(ctx, "assistant")
		query += "\n\n" + convertRole
	}
	return
}

func genToken(ctx context.Context, ident map[string]string, proxied, nTok bool) (accessToken string, err error) {
	cookie := ident["cookie"]
	scopeId := ident["scopeId"]
	cacheManager := cache.BingCacheManager()
	accessToken, _ = cacheManager.GetValue(cookie)
	if !nTok && accessToken != "" {
		accessToken = strings.Split(accessToken, "|")[1]
		return
	}

	mu.Lock()
	defer mu.Unlock()

	idToken, ok := ident["idToken"]
	if !ok {
		err = fmt.Errorf("invalid jwt")
		return
	}

	token, _ := jwt.Parse(idToken, func(token *jwt.Token) (zero interface{}, err error) { return })
	if token == nil {
		err = fmt.Errorf("invalid jwt")
		return
	}
	claims := token.Claims.(jwt.MapClaims)

	if nTok || accessToken == "" {
		accessToken, err = edge.Authorize(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), ctx, scopeId, idToken, cookie)
	} else {
		accessToken, err = edge.RefreshToken(elseOf(proxied, common.HTTPClient, common.NopHTTPClient), ctx, claims["aud"].(string), scopeId, strings.Split(accessToken, "|")[0])
	}
	if err != nil {
		return
	}

	err = cacheManager.SetWithExpiration(cookie, accessToken, time.Hour)
	accessToken = strings.Split(accessToken, "|")[1]
	return
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}
