package handler

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/bing"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"net/http"
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
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"message": err.Error(),
				},
			})
			return
		}

		switch chatCompletionRequest.Model {
		case "claude-2":
		case "bing":
			bing.Complete(ctx, token, proxies, chatCompletionRequest)
		case "gemini":
		}
	}
}
