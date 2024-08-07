package handler

import (
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"fmt"

	"github.com/gin-gonic/gin"
)

func embedding(ctx *gin.Context) {

	var embedding pkg.EmbedRequest
	if err := ctx.BindJSON(&embedding); err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}
	_ = ctx.Request.Body.Close()
	ctx.Set(vars.GinEmbedding, embedding)

	if !GlobalExtension.Match(ctx, embedding.Model) {
		response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", embedding.Model))
		return
	}

	GlobalExtension.Embedding(ctx)
}
