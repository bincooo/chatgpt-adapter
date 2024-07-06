package v1

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"net/http"
)

func fetch(ctx *gin.Context, proxies, token string, completion pkg.ChatCompletion) (response *http.Response, err error) {
	var (
		baseUrl  = pkg.Config.GetString("custom-llm.baseUrl")
		useProxy = pkg.Config.GetBool("custom-llm.useProxy")
	)

	if !useProxy {
		proxies = ""
	}

	if completion.TopP == 0 {
		completion.TopP = 1
	}

	if completion.Temperature == 0 {
		completion.Temperature = 0.7
	}

	if completion.MaxTokens == 0 {
		completion.MaxTokens = 1024
	}

	tokens := 0
	for _, message := range completion.Messages {
		tokens += common.CalcTokens(message.GetString("content"))
	}
	ctx.Set(ginTokens, token)

	completion.Stream = true
	response, err = emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(baseUrl+"/v1/chat/completions").
		Header("Authorization", "Bearer "+token).
		JHeader().
		Body(completion).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	return
}
