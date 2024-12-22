package v1

import (
	"encoding/json"
	"net/http"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
)

func fetch(ctx *gin.Context, proxies, token string, completion model.Completion) (r *http.Response, err error) {
	var (
		baseUrl = ctx.GetString(key)
	)

	if !ctx.GetBool(upKey) {
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
		tokens += response.CalcTokens(message.GetString("content"))
	}
	ctx.Set(ginTokens, token)

	completion.Stream = true
	completion.Model = ctx.GetString(modKey)
	obj, err := toMap(completion)
	if err != nil {
		return nil, err
	}

	if completion.TopK == 0 {
		delete(obj, "top_k")
	}

	r, err = emit.ClientBuilder(common.HTTPClient).
		Proxies(proxies).
		Context(ctx).
		POST(baseUrl+"/chat/completions").
		Header("Authorization", "Bearer "+token).
		JSONHeader().
		Body(obj).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	return
}

func toMap(obj interface{}) (mo map[string]interface{}, err error) {
	if obj == nil {
		return
	}

	bytes, err := json.Marshal(obj)
	if err != nil {
		return
	}

	err = json.Unmarshal(bytes, &mo)
	return
}
