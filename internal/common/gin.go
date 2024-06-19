package common

import (
	"context"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"time"
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
			"id":      "-1",
			"enabled": false,
			"tasks":   false,
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

func GetGinContext(ctx *gin.Context) context.Context {
	var key = "__context__"
	{
		value, exists := GetGinValue[context.Context](ctx, key)
		if exists {
			return value
		}
	}

	reqCtx := ctx.Request.Context()
	connTimeout := pkg.Config.GetInt("server-conn.connTimeout")
	if connTimeout > 0 {
		timeout, cancelFunc := context.WithTimeout(reqCtx, time.Duration(connTimeout)*time.Second)
		ctx.Set(key, timeout)
		ctx.Set(vars.GinCancelFunc, cancelFunc)
		return timeout
	}
	return reqCtx
}

func GetGinIdleConnectOption(ctx *gin.Context) *emit.ConnectOption {
	key := "__IdleConnectOption__"
	{
		value, exists := GetGinValue[*emit.ConnectOption](ctx, key)
		if exists {
			return value
		}
	}

	idleEnabled := false
	opts := pkg.Config.GetStringMap("server-conn")
	var option emit.ConnectOption
	if value, ok := opts["idleconntimeout"]; ok {
		connTimeout, o := value.(int)
		if o {
			if connTimeout > 0 {
				option.IdleConnTimeout = time.Duration(connTimeout) * time.Second
				idleEnabled = true
			}
		} else {
			logger.Warnf("read idleConnTimeout error: %v", value)
		}
	}

	if value, ok := opts["responseheadertimeout"]; ok {
		connTimeout, o := value.(int)
		if o {
			if connTimeout > 0 {
				option.ResponseHeaderTimeout = time.Duration(connTimeout) * time.Second
				idleEnabled = true
			}
		} else {
			logger.Warnf("read responseHeaderTimeout error: %v", value)
		}
	}

	if value, ok := opts["expectcontinuetimeout"]; ok {
		connTimeout, o := value.(int)
		if o {
			if connTimeout > 0 {
				option.ExpectContinueTimeout = time.Duration(connTimeout) * time.Second
				idleEnabled = true
			}
		} else {
			logger.Warnf("read expectContinueTimeout error: %v", value)
		}
	}

	if !idleEnabled {
		return nil
	}

	ctx.Set(key, &option)
	return &option
}
