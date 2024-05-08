package handler

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/bing"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/claude"
	coh "github.com/bincooo/chatgpt-adapter/v2/internal/middle/cohere"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/coze"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/gemini"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/lmsys"
	pg "github.com/bincooo/chatgpt-adapter/v2/internal/middle/playground"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/sd"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/cohere-api"
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

	// cohere 默认generate模式
	switch chatCompletionRequest.Model {
	case cohere.COMMAND,
		cohere.COMMAND_R,
		cohere.COMMAND_LIGHT,
		cohere.COMMAND_LIGHT_NIGHTLY,
		cohere.COMMAND_NIGHTLY,
		cohere.COMMAND_R_PLUS:
		ctx.Set("notebook", true)
	}

	matchers := common.XmlFlags(ctx, &chatCompletionRequest)
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
	case "gemini-1.0":
		gemini.Complete(ctx, chatCompletionRequest, matchers)
	case "gemini-1.5":
		token := ctx.GetString("token")
		if strings.HasPrefix(token, "AIzaSy") {
			gemini.Complete(ctx, chatCompletionRequest, matchers)
		} else {
			gemini.Complete15(ctx, chatCompletionRequest, matchers)
		}
	case "coze":
		coze.Complete(ctx, chatCompletionRequest, matchers)
	case cohere.COMMAND,
		cohere.COMMAND_R,
		cohere.COMMAND_LIGHT,
		cohere.COMMAND_LIGHT_NIGHTLY,
		cohere.COMMAND_NIGHTLY,
		cohere.COMMAND_R_PLUS:
		coh.Complete(ctx, chatCompletionRequest, matchers)
	default:
		if strings.HasPrefix(chatCompletionRequest.Model, "lmsys/") {
			lmsys.Complete(ctx, chatCompletionRequest, matchers)
			return
		}

		if strings.HasPrefix(chatCompletionRequest.Model, "claude-") {
			claude.Complete(ctx, chatCompletionRequest, matchers)
			return
		}

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
	if strings.Contains(token, "msToken=") || strings.Contains(token, "sessionid=") {
		chatGenerationRequest.Model = "coze." + chatGenerationRequest.Model
	} else if token == "sk-prodia-xl" {
		ctx.Set("prodia.space", "xl")
		chatGenerationRequest.Model = "xl." + chatGenerationRequest.Model
	} else if token == "sk-prodia-sd" {
		ctx.Set("prodia.space", "sd")
		chatGenerationRequest.Model = "sd." + chatGenerationRequest.Model
	} else if token == "sk-krebzonide" {
		ctx.Set("prodia.space", "kb")
		chatGenerationRequest.Model = "kb." + chatGenerationRequest.Model
	} else if ok, _ := regexp.MatchString(`\w{8,10}-\w{4}-\w{4}-\w{4}-\w{10,15}`, token); ok {
		chatGenerationRequest.Model = "pg." + chatGenerationRequest.Model
	} else {
		chatGenerationRequest.Model = "sd." + chatGenerationRequest.Model
	}

	logrus.Infof("generate images text[ %s ]: %s", chatGenerationRequest.Model, chatGenerationRequest.Prompt)
	switch chatGenerationRequest.Model {
	case "coze.dall-e-3":
		coze.Generation(ctx, chatGenerationRequest)
	case "xl.dall-e-3", "sd.dall-e-3", "kb.dall-e-3", "dall-e-3":
		sd.Generation(ctx, chatGenerationRequest)
	case "pg.dall-e-3":
		pg.Generation(ctx, chatGenerationRequest)
	default:
		middle.ResponseWithV(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", chatGenerationRequest.Model))
	}
}
