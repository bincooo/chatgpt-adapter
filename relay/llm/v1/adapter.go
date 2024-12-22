package v1

import (
	"net/http"
	"strings"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

var (
	Model  = "custom"
	schema = make([]map[string]interface{}, 0)
	key    = "__custom-url__"
	upKey  = "__custom-proxies__"
	modKey = "__custom-model__"
	tcKey  = "__custom-toolCall__"
)

type api struct {
	inter.BaseAdapter
	env *env.Environment
}

func init() {
	inited.AddInitialized(func(env *env.Environment) {
		llm := env.Get("custom-llm")
		if slice, ok := llm.([]interface{}); ok {
			for _, it := range slice {
				item, o := it.(map[string]interface{})
				if !o {
					continue
				}
				schema = append(schema, item)
			}
		}
	})
}

func (*api) Match(ctx *gin.Context, model string) (ok bool, _ error) {
	for _, it := range schema {
		if prefix, o := it["prefix"].(string); o && strings.HasPrefix(model, prefix+"/") {
			ctx.Set(key, it["reversal"])
			ctx.Set(upKey, it["proxied"] == "true")
			ctx.Set(modKey, model[len(prefix)+1:])
			ctx.Set(tcKey, it["tc"] == "true")
			ok = true
			return
		}
	}
	return
}

func (*api) Models() []model.Model {
	return []model.Model{
		{
			Id:      "custom",
			Object:  "model",
			Created: 1686935002,
			By:      "custom-adapter",
		},
	}
}

func (api *api) ToolChoice(ctx *gin.Context) (ok bool, err error) {
	var (
		proxies    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)
	if !ctx.GetBool(tcKey) {
		return
	}
	if toolChoice(ctx, proxies, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	r, err := fetch(ctx, proxies, cookie, completion)
	if err != nil {
		logger.Error(err)
		return
	}

	defer r.Body.Close()
	content := waitResponse(ctx, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		logger.Error("EMPTY RESPONSE")
	}
	return
}

func (api *api) Embedding(ctx *gin.Context) (err error) {
	embedding := common.GetGinEmbedding(ctx)
	embedding.Model = ctx.GetString(modKey)
	var (
		token   = ctx.GetString("token")
		proxies = api.env.GetString("proxied")
		baseUrl = ctx.GetString(key)
	)
	if !ctx.GetBool(upKey) {
		proxies = ""
	}

	resp, err := emit.ClientBuilder(common.HTTPClient).
		Proxies(proxies).
		Context(ctx).
		POST(baseUrl+"/embeddings").
		Header("Authorization", "Bearer "+token).
		JSONHeader().
		Body(embedding).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		logger.Error(err)
		return
	}

	obj, err := emit.ToMap(resp)
	if err != nil {
		logger.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, obj)
	return
}
