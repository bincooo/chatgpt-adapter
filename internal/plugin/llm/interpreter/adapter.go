package interpreter

import (
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/gin-gonic/gin"
)

// OpenInterpreter/open-interpreter
var (
	Adapter = API{}
	Model   = "open-interpreter"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	return model == Model
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "open-interpreter",
			Object:  "model",
			Created: 1686935002,
			By:      "interpreter-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	r, tokens, err := fetch(ctx, proxies, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	ctx.Set(ginTokens, tokens)
	content := waitResponse(ctx, matchers, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}
