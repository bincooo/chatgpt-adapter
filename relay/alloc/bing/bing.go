package bing

import (
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/proxy"
	"github.com/iocgo/sdk/stream"
)

var (
	cookiesContainer *common.PollContainer[map[string]string]
)

func init() {
	inited.AddInitialized(func(env *env.Environment) {
		cookies, ok := env.Get("bing.cookies").([]interface{})
		if !ok {
			return
		}
		slice := stream.Map(stream.OfSlice(cookies), func(t interface{}) (obj map[string]string) {
			m, o := t.(map[string]interface{})
			if !o {
				return
			}
			return map[string]string{
				"scopeId": m["scopeid"].(string),
				"idToken": m["idtoken"].(string),
				"cookie":  m["cookie"].(string),
			}
		}).ToSlice()

		cookiesContainer = common.NewPollContainer[map[string]string]("bing", slice, 6*time.Hour)
		cookiesContainer.Condition = condition
	})
}

func InvocationHandler(ctx *proxy.Context) {
	var (
		gtx  = ctx.In[0].(*gin.Context)
		echo = gtx.GetBool(vars.GinEcho)
	)

	if echo || ctx.Method != "Completion" && ctx.Method != "ToolChoice" {
		ctx.Do()
		return
	}

	logger.Infof("execute static proxy [relay/llm/bing.api]: func %s(...)", ctx.Method)

	if cookiesContainer.Len() == 0 {
		response.Error(gtx, -1, "empty cookies")
		return
	}

	cookie, err := cookiesContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(gtx, -1, err)
		return
	}
	defer resetMarked(cookie)
	gtx.Set("token", cookie)

	//
	ctx.Do()

	//
	if ctx.Method == "Completion" {
		err = elseOf[error](ctx.Out[0])
	}
	if ctx.Method == "ToolChoice" {
		err = elseOf[error](ctx.Out[1])
	}

	if err != nil {
		logger.Error(err)
		return
	}
}

func condition(cookie map[string]string, argv ...interface{}) bool {
	marker, err := cookiesContainer.Marked(cookie)
	if err != nil {
		logger.Error(err)
		return false
	}
	return marker == 0
}

func resetMarked(cookie map[string]string) {
	marker, err := cookiesContainer.Marked(cookie)
	if err != nil {
		logger.Error(err)
		return
	}

	if marker != 1 {
		return
	}

	err = cookiesContainer.MarkTo(cookie, 0)
	if err != nil {
		logger.Error(err)
	}
}

func elseOf[T any](obj any) (zero T) {
	if obj == nil {
		return
	}
	return obj.(T)
}
