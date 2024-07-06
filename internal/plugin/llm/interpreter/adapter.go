package interpreter

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
	socketio "github.com/zishang520/socket.io/socket"
	"sync"
)

// OpenInterpreter/open-interpreter
var (
	Adapter = API{}
	Model   = "open-interpreter"

	mu     sync.Mutex
	ws     *socketio.Socket
	wsChan chan string
)

type API struct {
	plugin.BaseAdapter
}

func init() {
	common.AddInitialized(func() {
		if !pkg.Config.GetBool("interpreter.ws") {
			return
		}

		err := plugin.IO.On("connection", func(events ...any) {
			if len(events) == 0 {
				return
			}

			w, ok := events[0].(*socketio.Socket)
			if !ok {
				return
			}

			mu.Lock()
			defer mu.Unlock()
			if !initSocketIO(w) {
				w.Disconnect(true)
				return
			}
			logger.Infof("connection event: %v", w)
		})
		if err != nil {
			logger.Errorf("socket.io connection event error: %v", err)
		}
	})
}

func (API) Match(_ *gin.Context, model string) bool {
	return model == Model || model == Model+"-ws"
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "open-interpreter",
			Object:  "model",
			Created: 1686935002,
			By:      "interpreter-adapter",
		},
		{
			Id:      "open-interpreter-ws",
			Object:  "model",
			Created: 1686935002,
			By:      "interpreter-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	if completion.Model == Model+"-ws" {
		completionWS(ctx)
		return
	}

	r, tokens, err := fetch(ctx, proxies, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	ctx.Set(ginTokens, tokens)
	content := waitResponse(ctx, matchers, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}
