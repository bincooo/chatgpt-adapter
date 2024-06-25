package plugin

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"time"
)

var (
	HTTPClient *emit.Session
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

		connTimeout := time.Duration(pkg.Config.GetInt("server-conn.connTimeout")) * time.Second
		if connTimeout == 0 {
			connTimeout = 180 * time.Second
		}

		HTTPJa3Client := emit.NewJa3Session(vars.Proxies, connTimeout)
		if err != nil {
			logger.Error("Error initializing HTTPJa3Client: ", err)
		}

		SocketClient, err := emit.NewSocketSession(vars.Proxies, option, whites...)
		if err != nil {
			logger.Error("Error initializing HTTPJa3Client: ", err)
		}

		HTTPClient = emit.MergeSession(HTTPClient, HTTPJa3Client, SocketClient)
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

func (adapter ExtensionAdapter) Generation(ctx *gin.Context) {
	completion := common.GetGinGeneration(ctx)
	for _, extension := range adapter.Extensions {
		if extension.Match(ctx, completion.Model) {
			extension.Generation(ctx)
			return
		}
	}
}
