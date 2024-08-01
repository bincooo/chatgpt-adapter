package v1

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"github.com/gin-gonic/gin"
	"strings"
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
