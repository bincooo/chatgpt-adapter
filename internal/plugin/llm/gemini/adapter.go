package gemini

import (
	"errors"
	com "github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/gin-gonic/gin"
	"net/url"
	"strings"
	"sync"
)

const MODEL = "gemini"
const login = "http://127.0.0.1:8081/v1/login"

var (
	Adapter = API{}
	mu      sync.Mutex
)

type cookieOpts struct {
	userAgent string
	cookie    string
}

type candidatesResponse struct {
	Candidates []candidate `json:"candidates"`
}

type candidate struct {
	Content struct {
		Role  string                   `json:"role"`
		Parts []map[string]interface{} `json:"parts"`
	} `json:"content"`
	FinishReason string `json:"finishReason"`
	Index        int    `json:"index"`
}

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case "gemini-1.0-pro-latest", "gemini-1.5-pro-latest", "gemini-1.5-flash-latest":
		return true
	default:
		return false
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "gemini-1.0-pro-latest",
			Object:  "model",
			Created: 1686935002,
			By:      "gemini-adapter",
		}, {
			Id:      "gemini-1.5-pro-latest",
			Object:  "model",
			Created: 1686935002,
			By:      "gemini-adapter",
		}, {
			Id:      "gemini-1.5-flash-latest",
			Object:  "model",
			Created: 1686935002,
			By:      "gemini-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		completion = com.GetGinCompletion(ctx)
		matchers   = com.GetGinMatchers(ctx)
	)

	newMessages, tokens, err := mergeMessages(completion.Messages)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	ctx.Set(ginTokens, tokens)
	r, err := build(ctx.Request.Context(), proxies, cookie, newMessages, completion)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) {
			urlError.URL = strings.ReplaceAll(urlError.URL, cookie, "AIzaSy***")
		}
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	// 最近似乎很容易发送空消息？
	content := waitResponse(ctx, matchers, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}
