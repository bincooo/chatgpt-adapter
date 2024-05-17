package common

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
)

func GetGinCompletion(ctx *gin.Context) pkg.ChatCompletion {
	obj, _ := ctx.Get(vars.GinCompletion)
	return obj.(pkg.ChatCompletion)
}

func GetGinGeneration(ctx *gin.Context) pkg.ChatGeneration {
	obj, _ := ctx.Get(vars.GinGeneration)
	return obj.(pkg.ChatGeneration)
}

func GetGinMatchers(ctx *gin.Context) []pkg.Matcher {
	obj, _ := ctx.Get(vars.GinMatchers)
	return obj.([]pkg.Matcher)
}
