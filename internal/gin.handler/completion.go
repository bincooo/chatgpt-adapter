package handler

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/hf"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/llm/bing"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/llm/claude"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/llm/cohere"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/llm/coze"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/llm/gemini"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/llm/lmsys"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/llm/v1"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/playground"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	GlobalExtension = middle.ExtensionAdapter{
		Extensions: make([]middle.Adapter, 0),
	}
)

func InitExtensions() {
	GlobalExtension.Extensions = []middle.Adapter{
		bing.Adapter,
		claude.Adapter,
		coh.Adapter,
		coze.Adapter,
		gemini.Adapter,
		lmsys.Adapter,
		pg.Adapter,
		hf.Adapter,
		v1.Adapter,
	}
}

func completions(ctx *gin.Context) {
	var completion pkg.ChatCompletion
	if err := ctx.BindJSON(&completion); err != nil {
		middle.ErrResponse(ctx, -1, err)
		return
	}
	ctx.Set(vars.GinCompletion, completion)
	matchers := common.XmlFlags(ctx, &completion)
	ctx.Set(vars.GinMatchers, matchers)
	if ctx.GetBool("debug") {
		indent, err := json.MarshalIndent(completion, "", "  ")
		if err != nil {
			logrus.Warn(err)
		} else {
			fmt.Printf("requset: \n%s", indent)
		}
	}

	if !middle.MessageValidator(ctx) {
		return
	}

	if !GlobalExtension.Match(ctx, completion.Model) {
		middle.ErrResponse(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", completion.Model))
		return
	}

	GlobalExtension.Completion(ctx)
}

func generations(ctx *gin.Context) {
	var generation pkg.ChatGeneration
	if err := ctx.BindJSON(&generation); err != nil {
		middle.ErrResponse(ctx, -1, err)
		return
	}

	ctx.Set(vars.GinGeneration, generation)
	logrus.Infof("generate images text[ %s ]: %s", generation.Model, generation.Message)
	if !GlobalExtension.Match(ctx, generation.Model) {
		middle.ErrResponse(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", generation.Model))
		return
	}

	GlobalExtension.Generation(ctx)
}
