package v1

import (
	"context"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"net/http"
)

func fetch(ctx context.Context, proxies string, completion pkg.ChatCompletion) (*http.Response, error) {
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

	completion.Stream = true
	return emit.ClientBuilder().
		Context(ctx).
		//Proxies(proxies).
		POST(baseUrl+"/v1/chat/completions").
		JHeader().
		Body(completion).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
}
