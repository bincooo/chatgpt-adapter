package plugin

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	socketio "github.com/zishang520/socket.io/socket"
)

var (
	HTTPClient     *emit.Session
	ClashAPIClient *emit.Session

	IO *socketio.Server
)

type Model struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	By      string `json:"owned_by"`
}

func init() {
	common.AddInitialized(func() {
		var err error
		whites := []string{
			"127.0.0.1",
		}

		option := common.GetIdleConnectOption()
		HTTPClient, err = emit.NewDefaultSession(vars.Proxies, option, whites...)
		if err != nil {
			logger.Error("Error initializing HTTPClient: ", err)
		}

		connTimeout := pkg.Config.GetInt("server-conn.connTimeout")
		if connTimeout == 0 {
			connTimeout = 180
		}

		HTTPJa3Client, err := emit.NewJa3Session(vars.Proxies, connTimeout)
		if err != nil {
			logger.Error("Error initializing HTTPJa3Client: ", err)
		}

		SocketClient, err := emit.NewSocketSession(vars.Proxies, option, whites...)
		if err != nil {
			logger.Error("Error initializing HTTPJa3Client: ", err)
		}

		HTTPClient = emit.MergeSession(HTTPClient, HTTPJa3Client, SocketClient)
		IO = socketio.NewServer(nil, nil)

		if value := pkg.Config.GetString("clash.proxies"); value != "" {
			ClashAPIClient, err = emit.NewDefaultSession(value, option, whites...)
			if err != nil {
				logger.Error("Error initializing ClashAPIClient: ", err)
			}
		}
	})
}

type Adapter interface {
	Match(ctx *gin.Context, model string) bool
	Models() []Model
	Completion(ctx *gin.Context)
	Generation(ctx *gin.Context)
}

type BaseAdapter struct {
}

type ExtensionAdapter struct {
	Extensions []Adapter
}

func (BaseAdapter) Models() []Model {
	return nil
}

func (BaseAdapter) Completion(*gin.Context) {
}

func (BaseAdapter) Generation(*gin.Context) {
}

func (adapter ExtensionAdapter) Match(ctx *gin.Context, model string) bool {
	for _, extension := range adapter.Extensions {
		if extension.Match(ctx, model) {
			return true
		}
	}
	return false
}

func (adapter ExtensionAdapter) Models() (models []Model) {
	for _, extension := range adapter.Extensions {
		models = append(models, extension.Models()...)
	}
	return
}

func (adapter ExtensionAdapter) Completion(ctx *gin.Context) {
	completion := common.GetGinCompletion(ctx)
	for _, extension := range adapter.Extensions {
		if extension.Match(ctx, completion.Model) {
			extension.Completion(ctx)
			return
		}
	}
	response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", completion.Model))
}

func (adapter ExtensionAdapter) Messages(ctx *gin.Context) {
	completion := common.GetGinCompletion(ctx)
	for _, extension := range adapter.Extensions {
		if extension.Match(ctx, completion.Model) {
			exec, ok := extension.(interface{ Messages(ctx *gin.Context) })
			if ok {
				exec.Messages(ctx)
				return
			}
		}
	}
	response.Error(ctx, -1, fmt.Sprintf("model '%s' is not not yet supported", completion.Model))
}

func (adapter ExtensionAdapter) Generation(ctx *gin.Context) {
	completion := common.GetGinGeneration(ctx)
	for _, extension := range adapter.Extensions {
		if extension.Match(ctx, completion.Model) {
			extension.Generation(ctx)
			return
		}
	}
}
