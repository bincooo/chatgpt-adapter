package common

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
)

func GinDebugger(ctx *gin.Context) bool {
	if value, ok := GetGinValue[bool](ctx, vars.GinDebugger); ok {
		return value
	}

	return false
}

func GetGinCompletion(ctx *gin.Context) (value pkg.ChatCompletion) {
	value, _ = GetGinValue[pkg.ChatCompletion](ctx, vars.GinCompletion)
	return
}

func GetGinGeneration(ctx *gin.Context) (value pkg.ChatGeneration) {
	value, _ = GetGinValue[pkg.ChatGeneration](ctx, vars.GinGeneration)
	return
}

func GetGinMatchers(ctx *gin.Context) (values []Matcher) {
	values, _ = GetGinValues[Matcher](ctx, vars.GinMatchers)
	return
}

func GetGinCompletionUsage(ctx *gin.Context) map[string]int {
	obj, exists := ctx.Get(vars.GinCompletionUsage)
	if exists {
		return obj.(map[string]int)
	}
	return nil
}

func GetGinToolValue(ctx *gin.Context) pkg.Keyv[interface{}] {
	tool, ok := GetGinValue[pkg.Keyv[interface{}]](ctx, vars.GinTool)
	if !ok {
		tool = pkg.Keyv[interface{}]{
			"id": "-1",
		}
	}
	return tool
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
