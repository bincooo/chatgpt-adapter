package plugin

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/gin-gonic/gin"
)

type Model struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	By      string `json:"owned_by"`
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
	completion.Model = "coze"
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
