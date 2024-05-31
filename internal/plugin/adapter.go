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
	switch completion.Model {
		case "gpt-3.5-turbo":
			//model = "coze/7372633086053482504-1716578081-2"
			completion.Model = "coze/7372633086053482504-7372633764939366401-2-o"
			break
		case "gpt-4o":
			//model = "coze/7372633086053482504-1716578081-2"
			completion.Model = "coze/7372633086053482504-7372633764939366401-2-o"
			break
		case "gpt-4":
			//model = "coze/7372646846499487751-1716578547-2"
			completion.Model = "coze/7372646846499487751-7372633764939366401-2-o"
			break
		case "gpt-4-turbo":
			//model = "coze/7372648254925930514-1716579154-2"
			completion.Model = "coze/7372648254925930514-7372633764939366401-2-o"
			break
		case "gemini-1.5-pro-lastest":
			//model = "coze/7372931363038625800-1716644517-2"
			completion.Model = "coze/7372931363038625800-7372633764939366401-2-o"
			break
	}
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
