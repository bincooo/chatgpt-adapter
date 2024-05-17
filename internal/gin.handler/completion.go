package handler

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/bing"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/claude"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/cohere"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/coze"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/gemini"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/lmsys"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/playground"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle/sd"
	v1 "github.com/bincooo/chatgpt-adapter/v2/internal/middle/v1"
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
		sd.Adapter,
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

	ctx.Set(vars.GinCompletion, generation)
	logrus.Infof("generate images text[ %s ]: %s", generation.Model, generation.Prompt)
	if !GlobalExtension.Match(ctx, generation.Model) {
		middle.ErrResponse(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", generation.Model))
		return
	}

	GlobalExtension.Generation(ctx)
}
