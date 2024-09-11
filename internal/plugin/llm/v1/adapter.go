package v1

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"net/http"
	"strings"

	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
)

var (
	Adapter = API{}
	Model   = "custom"
	schema  = make([]map[string]interface{}, 0)
	key     = "__custom-url__"
	upKey   = "__custom-proxies__"
	modKey  = "__custom-model__"
	tcKey   = "__custom-toolCall__"
)

type API struct {
	plugin.BaseAdapter
}

func init() {
	common.AddInitialized(func() {
		llm := pkg.Config.Get("custom-llm")
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

func (API) Match(ctx *gin.Context, model string) bool {
	for _, it := range schema {
		if prefix, ok := it["prefix"].(string); ok && strings.HasPrefix(model, prefix+"/") {
			ctx.Set(key, it["base-url"])
			ctx.Set(upKey, it["use-proxies"] == "true")
			ctx.Set(modKey, model[len(prefix)+1:])
			ctx.Set(tcKey, it["tc"] == "true")
			return true
		}
	}
	return false
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "custom",
			Object:  "model",
			Created: 1686935002,
			By:      "custom-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	completion.Model = completion.Model[7:]
	if ctx.GetBool(tcKey) && plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, proxies, completion) {
			return
		}
	}

	retry := 3
label:
	r, err := fetch(ctx, proxies, cookie, completion)
	if err != nil {
		if retry > 0 {
			retry--
			goto label
		}

		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	defer r.Body.Close()
	content := waitResponse(ctx, matchers, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func (API) Embedding(ctx *gin.Context) {
	embedding := common.GetGinEmbedding(ctx)
	embedding.Model = ctx.GetString(modKey)
	var (
		token   = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
		baseUrl = ctx.GetString(key)
	)
	if !ctx.GetBool(upKey) {
		proxies = ""
	}

	resp, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(baseUrl+"/embeddings").
		Header("Authorization", "Bearer "+token).
		JHeader().
		Body(embedding).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		response.Error(ctx, http.StatusBadGateway, err)
		return
	}

	obj, err := emit.ToMap(resp)
	if err != nil {
		response.Error(ctx, http.StatusBadGateway, err)
		return
	}

	ctx.JSON(http.StatusOK, obj)
}
