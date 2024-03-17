package handler

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/bing"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/claude"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/coze"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/gemini"
	pg "github.com/bincooo/chatgpt-adapter/v2/internal/middle/playground"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/sd"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

func completions(ctx *gin.Context) {
	var chatCompletionRequest gpt.ChatCompletionRequest
	if err := ctx.BindJSON(&chatCompletionRequest); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	common.XmlPlot(ctx, chatCompletionRequest.Messages)
	matchers := common.NewMatchers()

	if ctx.GetBool("debug") {
		indent, err := json.MarshalIndent(chatCompletionRequest, "", "  ")
		if err != nil {
			logrus.Warn(err)
		} else {
			fmt.Printf("requset: \n%s", indent)
		}
	}

	switch chatCompletionRequest.Model {
	case "bing":
		bing.Complete(ctx, chatCompletionRequest, matchers)
	case "claude":
		claude.Complete(ctx, chatCompletionRequest, matchers)
	case "gemini":
		gemini.Complete(ctx, chatCompletionRequest, matchers)
	case "coze":
		coze.Complete(ctx, chatCompletionRequest, matchers)
	default:
		if strings.HasPrefix(chatCompletionRequest.Model, "claude-") {
			claude.Complete(ctx, chatCompletionRequest, matchers)
		} else {
			middle.ResponseWithV(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", chatCompletionRequest.Model))
		}
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
	} else if ok, _ := regexp.MatchString(`\w{8,10}-\w{4}-\w{4}-\w{4}-\w{10,15}`, token); ok {
		chatGenerationRequest.Model = "pg." + chatGenerationRequest.Model
	} else {
		chatGenerationRequest.Model = "sd." + chatGenerationRequest.Model
	}

	switch chatGenerationRequest.Model {
	//case "bing.dall-e-3":
	// oneapi目前只认dall-e-3
	case "coze.dall-e-3":
		coze.Generation(ctx, chatGenerationRequest)
	case "sd.dall-e-3":
		sd.Generation(ctx, chatGenerationRequest)
	case "pg.dall-e-3":
		pg.Generation(ctx, chatGenerationRequest)
	default:
		middle.ResponseWithV(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", chatGenerationRequest.Model))
	}
}
