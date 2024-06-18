package handler

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/hf"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/bing"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/claude"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/cohere"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/coze"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/gemini"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/interpreter"
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/lmsys"
	v1 "github.com/bincooo/chatgpt-adapter/internal/plugin/llm/v1"
	pg "github.com/bincooo/chatgpt-adapter/internal/plugin/playground"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
	"math/rand"
	"time"
)

var (
	GlobalExtension = plugin.ExtensionAdapter{
		Extensions: make([]plugin.Adapter, 0),
	}
)

func init() {
	common.AddInitialized(func() {
		GlobalExtension.Extensions = []plugin.Adapter{
			bing.Adapter,
			claude.Adapter,
			cohere.Adapter,
			coze.Adapter,
			gemini.Adapter,
			interpreter.Adapter,
			lmsys.Adapter,
			pg.Adapter,
			hf.Adapter,
			v1.Adapter,
		}
	})
}

func completions(ctx *gin.Context) {
	var completion pkg.ChatCompletion
	if err := ctx.BindJSON(&completion); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	ctx.Set(vars.GinDebugger, pkg.Config.GetBool("debug"))
	toolCall := pkg.Config.GetStringMap("toolCall")
	if enabled, ok := toolCall["enabled"]; ok && enabled.(bool) {
		id := fmt.Sprintf("%v", toolCall["id"])
		if id == "" {
			id = "-1"
		}

		ctx.Set(vars.GinTool, pkg.Keyv[interface{}]{
			"id":      id,
			"enabled": enabled,
			"tasks":   toolCall["tasks"].(bool),
		})
	}

	matchers := common.XmlFlags(ctx, &completion)
	ctx.Set(vars.GinCompletion, completion)

	ctx.Set(vars.GinMatchers, matchers)
	if common.GinDebugger(ctx) {
		bodyLogger(completion)
	}

	if !response.MessageValidator(ctx) {
		return
	}

	if !GlobalExtension.Match(ctx, completion.Model) {
		response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", completion.Model))
		return
	}

	GlobalExtension.Completion(ctx)
}

func generations(ctx *gin.Context) {
	var generation pkg.ChatGeneration
	if err := ctx.BindJSON(&generation); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	ctx.Set(vars.GinGeneration, generation)
	logger.Infof("generate images text[ %s ]: %s", generation.Model, generation.Message)
	if !GlobalExtension.Match(ctx, generation.Model) {
		if generation.Model != "dall-e-3" {
			response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", generation.Model))
			return
		}

		// 默认适配一个
		tokens := []string{
			"sk-prodia-sd",
			"sk-prodia-xl",
			"sk-google-xl",
			"sk-dalle-4k",
			"sk-dalle-3-xl",
			"sk-animagine-xl-3.1",
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		ctx.Set("token", tokens[r.Intn(len(tokens)-1)])
		if !GlobalExtension.Match(ctx, generation.Model) {
			response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", generation.Model))
			return
		}
	}

	GlobalExtension.Generation(ctx)
}

func bodyLogger(completion pkg.ChatCompletion) {
	bytes, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		logger.Error(err)
	} else {
		logger.Infof("requset: \n%s", bytes)
	}
}
