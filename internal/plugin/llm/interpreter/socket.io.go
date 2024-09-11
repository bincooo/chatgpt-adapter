package interpreter

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
	socketio "github.com/zishang520/socket.io/socket"
)

func completionWS(ctx *gin.Context) {
	if ws == nil {
		response.Error(ctx, -1, "socket.io connection is closed / disabled")
		return
	}

	var (
		baseUrl    = pkg.Config.GetString("interpreter.base-url")
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	tokens, message, err := mergeMessages(ctx, proxies, baseUrl, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}
	ctx.Set(ginTokens, tokens)

	err = ws.Emit("reply", message)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	wsChan = make(chan string)
	content := waitResponseWS(ctx, matchers, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func initSocketIO(w *socketio.Socket) bool {
	if ws != nil {
		return false
	}

	r := w.Request()
	if token := r.GetPathInfo(); token != "/socket.io/open-i/" {
		return false
	}

	w.On("disconnect", func(...any) {
		mu.Lock()
		defer mu.Unlock()
		ws = nil
	})

	w.On("ping", func(...any) {
		w.Emit("pong", "ok")
	})

	w.On("message", func(args ...any) {
		message := args[0].(string)
		// TODO -
		if ws != nil {
			wsChan <- message
		}
		logger.Infof("message: %s", message)
		w.Emit("message", "ok")
	})

	ws = w
	return true
}
