package v1

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"io"
	"net/http"
	"strings"

	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
)

var (
	Adapter = API{}
	Model   = "custom"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	return strings.HasPrefix(model, "custom/")
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "custom",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
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
	if plugin.NeedToToolCall(ctx) {
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
	embedding.Model = embedding.Model[7:]
	var (
		token    = ctx.GetString("token")
		proxies  = ctx.GetString("proxies")
		baseUrl  = pkg.Config.GetString("custom-llm.baseUrl")
		useProxy = pkg.Config.GetBool("custom-llm.useProxy")
	)
	if !useProxy {
		proxies = ""
	}
	resp, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(baseUrl+"/v1/embeddings").
		Header("Authorization", "Bearer "+token).
		JHeader().
		Body(embedding).DoC(emit.Status(http.StatusOK))
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error": "can't send request to upstream",
		})
	}
	ctx.Header("Content-Type", "application/json; charset=utf-8")
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error": "can't read from upstream",
		})
	}
	ctx.Writer.Write(content)
	ctx.Writer.Flush()
}
