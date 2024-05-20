package common

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
)

func GetGinCompletion(ctx *gin.Context) (value pkg.ChatCompletion) {
	value, _ = GetGinValue[pkg.ChatCompletion](ctx, vars.GinCompletion)
	return
}

func GetGinGeneration(ctx *gin.Context) (value pkg.ChatGeneration) {
	value, _ = GetGinValue[pkg.ChatGeneration](ctx, vars.GinGeneration)
	return
}

func GetGinMatchers(ctx *gin.Context) (values []pkg.Matcher) {
	values, _ = GetGinValues[pkg.Matcher](ctx, vars.GinMatchers)
	return
}

func GetGinCompletionUsage(ctx *gin.Context) map[string]int {
	obj, exists := ctx.Get(vars.GinCompletionUsage)
	if exists {
		return obj.(map[string]int)
	}
	return nil
}

func GetGinValue[T any](ctx *gin.Context, key string) (t T, ok bool) {
	value, exists := ctx.Get(key)
	if !exists {
		return
	}

	t, ok = value.(T)
	return
}

func GetGinValues[T any](ctx *gin.Context, key string) ([]T, bool) {
	value, exists := ctx.Get(key)
	if !exists {
		return nil, false
	}

	t, ok := value.([]T)
	return t, ok
}
