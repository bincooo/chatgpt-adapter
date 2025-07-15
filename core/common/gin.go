package common

import (
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"github.com/gin-gonic/gin"
)

func GetGinCompletion(ctx *gin.Context) (value model.Completion) {
	value, _ = GetGinValue[model.Completion](ctx, vars.GinCompletion)
	return
}

func GetGinEmbedding(ctx *gin.Context) (value model.Embed) {
	value, _ = GetGinValue[model.Embed](ctx, vars.GinEmbedding)
	return
}

func GetGinGeneration(ctx *gin.Context) (value model.Generation) {
	value, _ = GetGinValue[model.Generation](ctx, vars.GinGeneration)
	return
}

func GetGinMatchers(ctx *gin.Context) (values []inter.Matcher) {
	values, _ = GetGinValues[inter.Matcher](ctx, vars.GinMatchers)
	return
}

func GetGinCompletionUsage(ctx *gin.Context) map[string]interface{} {
	obj, exists := ctx.Get(vars.GinCompletionUsage)
	if exists {
		return obj.(map[string]interface{})
	}
	return nil
}

func GetGinToolValue(ctx *gin.Context) model.Keyv[interface{}] {
	tool, ok := GetGinValue[model.Keyv[interface{}]](ctx, vars.GinTool)
	if !ok {
		tool = model.Keyv[interface{}]{
			"id":      "-1",
			"enabled": false,
			"tasks":   false,
		}
	}
	return tool
}

func IsGinCozeWebsdk(ctx *gin.Context) bool {
	return ctx.GetBool(vars.GinCozeWebsdk)
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

// func GetGinContext(ctx *gin.context) context.context {
// 	var key = "__context__"
// 	{
// 		value, exists := GetGinValue[context.context](ctx, key)
// 		if exists {
// 			return value
// 		}
// 	}
//
// 	reqCtx := ctx.Request.context()
// 	connTimeout := gin2.Config.GetInt("server-conn.connTimeout")
// 	if connTimeout > 0 {
// 		timeout, cancelFunc := context.WithTimeout(reqCtx, time.Duration(connTimeout)*time.Second)
// 		ctx.Set(key, timeout)
// 		ctx.Set(vars.GinCancelFunc, cancelFunc)
// 		return timeout
// 	}
// 	return reqCtx
// }

// func GetGinIdleConnectOption(ctx *gin.context) *emit.ConnectOption {
//	key := "__IdleConnectOption__"
//	{
//		value, exists := GetGinValue[*emit.ConnectOption](ctx, key)
//		if exists {
//			return value
//		}
//	}
//
//	option := GetIdleConnectOption()
//	ctx.Set(key, option)
//	return option
// }
