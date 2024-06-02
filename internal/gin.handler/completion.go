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
	"github.com/bincooo/chatgpt-adapter/internal/plugin/llm/lmsys"
	v1 "github.com/bincooo/chatgpt-adapter/internal/plugin/llm/v1"
	pg "github.com/bincooo/chatgpt-adapter/internal/plugin/playground"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
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
	matchers := common.XmlFlags(ctx, &completion)
	ctx.Set(vars.GinCompletion, completion)
	completion.Model = "coze"
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
		response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", generation.Model))
		return
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
