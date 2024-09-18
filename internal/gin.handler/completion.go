package handler

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/plugin/hf"
	"chatgpt-adapter/internal/plugin/llm/bing"
	"chatgpt-adapter/internal/plugin/llm/claude"
	"chatgpt-adapter/internal/plugin/llm/cohere"
	"chatgpt-adapter/internal/plugin/llm/coze"
	"chatgpt-adapter/internal/plugin/llm/gemini"
	"chatgpt-adapter/internal/plugin/llm/interpreter"
	"chatgpt-adapter/internal/plugin/llm/lmsys"
	"chatgpt-adapter/internal/plugin/llm/v1"
	"chatgpt-adapter/internal/plugin/llm/you"
	pg "chatgpt-adapter/internal/plugin/playground"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"math/rand"
	"time"
)

var (
	GlobalExtension = plugin.NewGlobalAdapter()
)

func init() {
	common.AddInitialized(func() {
		GlobalExtension.Add(
			bing.Adapter,
			claude.Adapter,
			cohere.Adapter,
			coze.Adapter,
			gemini.Adapter,
			interpreter.Adapter,
			lmsys.Adapter,
			you.Adapter,
			pg.Adapter,
			hf.Adapter,
			v1.Adapter,
		)
	})
}

func messages(ctx *gin.Context) {
	var completion pkg.ChatCompletion
	if err := ctx.BindJSON(&completion); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	_ = ctx.Request.Body.Close()
	matchers := common.NewMatchers(func(string) {})
	ctx.Set(vars.GinCompletion, completion)
	ctx.Set(vars.GinMatchers, matchers)

	if !GlobalExtension.Match(ctx, completion.Model) {
		response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", completion.Model))
		return
	}

	defer afterProcess(ctx)
	GlobalExtension.Messages(ctx)
}

func completions(ctx *gin.Context) {
	var completion pkg.ChatCompletion
	if err := ctx.BindJSON(&completion); err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	_ = ctx.Request.Body.Close()
	err := beforeProcess(ctx, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

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

	defer afterProcess(ctx)
	GlobalExtension.Completion(ctx)
}

func generations(ctx *gin.Context) {
	var generation pkg.ChatGeneration
	if err := ctx.BindJSON(&generation); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	_ = ctx.Request.Body.Close()
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

	defer afterProcess(ctx)
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

func beforeProcess(ctx *gin.Context, completion pkg.ChatCompletion) (err error) {
	// init debug
	ctx.Set(vars.GinDebugger, pkg.Config.GetBool("debug"))

	// init toolCall
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

	// init flags
	matchers, err := common.XmlFlags(ctx, &completion, func(str string) {
		if completion.Stream {
			response.SSEResponse(ctx, "matcher", str, time.Now().Unix())
		}
	})
	if err != nil {
		return err
	}
	ctx.Set(vars.GinCompletion, completion)

	// init matchers
	ctx.Set(vars.GinMatchers, matchers)
	return
}

func afterProcess(ctx *gin.Context) {
	cancel, exist := common.GetGinValue[context.CancelFunc](ctx, vars.GinCancelFunc)
	if exist {
		cancel()
	}
}
