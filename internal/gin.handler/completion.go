package handler

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/bing"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/claude"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/coze"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/gemini"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
)

func completions(ctx *gin.Context) {
	var chatCompletionRequest gpt.ChatCompletionRequest
	if err := ctx.BindJSON(&chatCompletionRequest); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	switch chatCompletionRequest.Model {
	case "bing":
		bing.Complete(ctx, chatCompletionRequest)
	case "claude-2":
		claude.Complete(ctx, chatCompletionRequest)
	case "gemini":
		gemini.Complete(ctx, chatCompletionRequest)
	case "coze":
		coze.Complete(ctx, chatCompletionRequest)
	default:
		middle.ResponseWithV(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", chatCompletionRequest.Model))
	}
}

func generations(ctx *gin.Context) {
	var chatGenerationRequest gpt.ChatGenerationRequest
	if err := ctx.BindJSON(&chatGenerationRequest); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	switch chatGenerationRequest.Model {
	//case "bing.dall-e-3":
	// oneapi目前只认dall-e-3
	case "dall-e-3", "coze.dall-e-3":
		coze.Generation(ctx, chatGenerationRequest)
	default:
		middle.ResponseWithV(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", chatGenerationRequest.Model))
	}
}
