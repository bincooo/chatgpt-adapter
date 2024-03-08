package handler

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/bing"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/claude"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/coze"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/gemini"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/sd"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"strings"
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

	token := ctx.GetString("token")
	if strings.Contains(token, "[msToken") {
		chatGenerationRequest.Model = "coze." + chatGenerationRequest.Model
		//} else if strings.HasPrefix(token, "AIzaSy") {
		//	chatGenerationRequest.Model = "gemini." + chatGenerationRequest.Model
	} else {
		chatGenerationRequest.Model = "sd." + chatGenerationRequest.Model
	}

	switch chatGenerationRequest.Model {
	//case "bing.dall-e-3":
	// oneapi目前只认dall-e-3
	case "coze.dall-e-3":
		coze.Generation(ctx, chatGenerationRequest)
	case "sd.dall-e-3":
		ctx.Set("openai.model", pkg.Config.GetString("openai.model"))
		ctx.Set("openai.baseUrl", pkg.Config.GetString("openai.baseUrl"))
		ctx.Set("openai.token", pkg.Config.GetString("openai.token"))
		ctx.Set("sd.baseUrl", pkg.Config.GetString("sd.baseUrl"))
		ctx.Set("sd.template", pkg.Config.GetString("sd.template"))
		sd.Generation(ctx, chatGenerationRequest)
	default:
		middle.ResponseWithV(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", chatGenerationRequest.Model))
	}
}
