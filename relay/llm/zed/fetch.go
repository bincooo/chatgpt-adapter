package zed

import (
	"bytes"
	"chatgpt-adapter/core/cache"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"context"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"net/http"
	"strings"
	"time"
)

var (
	userAgent = "Zed/0.178.5 (macos; x86_64)"
)

type Float32 float32

func (f Float32) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.1f", f)), nil
}

type zedRequest struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ProviderRequest struct {
		Model       string                    `json:"model"`
		MaxTokens   int                       `json:"max_tokens"`
		Temperature Float32                   `json:"temperature"`
		System      string                    `json:"system"`
		Messages    []model.Keyv[interface{}] `json:"messages"`
	} `json:"provider_request"`
}

func fetch(ctx *gin.Context, env *env.Environment, proxied, cookie string, request zedRequest) (response *http.Response, err error) {
	token := env.GetString("zed.token")
	if token != "" {
		cookie = token
		if strings.HasPrefix(cookie, "http") {
			token, err = genToken(ctx.Request.Context(), token)
			if err != nil {
				return
			}
			cookie = token
		}
	}

	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(proxied).
		POST("https://llm.zed.dev/completion").
		JSONHeader().
		Header("accept", "*/*").
		Header("host", "llm.zed.dev").
		Header("user-agent", userAgent).
		Header("authorization", "Bearer "+cookie).
		Body(request).
		DoS(http.StatusOK)
	return
}

func genToken(ctx context.Context, url string) (value string, err error) {
	manager := cache.ZedCacheManager()
	if value, _ = manager.GetValue(url); value != "" {
		return
	}

	var resp *http.Response
	retry := 2
label:
	retry--
	resp, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		GET(url).DoS(http.StatusOK)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	value = emit.TextResponse(resp)
	if value == "" || !strings.HasPrefix(value, "Bearer ") {
		if retry > 0 {
			goto label
		}

		err = errors.New("invalid token")
		return
	}
	value = value[7:]
	_ = manager.SetWithExpiration(url, value, time.Hour)
	return
}

func convertRequest(completion model.Completion) (request zedRequest, err error) {
	contentBuffer := new(bytes.Buffer)
	customInstructions := ""

	if len(completion.Messages) == 1 {
		contentBuffer.WriteString(completion.Messages[0].GetString("content"))
		goto label
	}

	if len(completion.Messages) > 1 {
		message := completion.Messages[0]
		if message.Is("role", "system") {
			customInstructions = message.GetString("content")
			completion.Messages = completion.Messages[1:]
		}
	}

label:
	request = zedRequest{
		Provider: "anthropic",
		Model:    completion.Model[4:],
		ProviderRequest: struct {
			Model       string                    `json:"model"`
			MaxTokens   int                       `json:"max_tokens"`
			Temperature Float32                   `json:"temperature"`
			System      string                    `json:"system"`
			Messages    []model.Keyv[interface{}] `json:"messages"`
		}{
			Model:       completion.Model[4:],
			MaxTokens:   completion.MaxTokens,
			System:      customInstructions,
			Messages:    completion.Messages,
			Temperature: Float32(completion.Temperature),
		},
	}
	return
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}
