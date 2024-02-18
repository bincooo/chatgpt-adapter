package handler

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/bing"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/claude"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/gemini"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"strings"
)

func completions(proxies string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var chatCompletionRequest gpt.ChatCompletionRequest

		token := ctx.Request.Header.Get("X-Api-Key")
		if token == "" {
			token = strings.TrimPrefix(ctx.Request.Header.Get("Authorization"), "Bearer ")
		}

		if err := ctx.BindJSON(&chatCompletionRequest); err != nil {
			middle.ResponseWithE(ctx, err)
			return
		}

		switch chatCompletionRequest.Model {
		case "bing":
			bing.Complete(ctx, token, proxies, chatCompletionRequest)
		case "claude-2":
			claude.Complete(ctx, token, proxies, chatCompletionRequest)
		case "gemini":
			gemini.Complete(ctx, token, proxies, chatCompletionRequest)
		default:
			middle.ResponseWithV(ctx, fmt.Sprintf("model '%s' is not not yet supported", chatCompletionRequest.Model))
		}
	}
}
