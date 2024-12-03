package inter

import (
	"chatgpt-adapter/core/gin/model"
	"github.com/gin-gonic/gin"
)

type Adapter interface {
	Match(ctx *gin.Context, model string) (bool, error)
	Models() []model.Model
	Completion(ctx *gin.Context) error
	Generation(ctx *gin.Context) error
	Embedding(ctx *gin.Context) error
	ToolChoice(ctx *gin.Context) (bool, error)
	HandleMessages(ctx *gin.Context, completion model.Completion) (messages []model.Keyv[interface{}], err error)
}

type BaseAdapter struct{}

func (BaseAdapter) Models() (slice []model.Model)                { return }
func (BaseAdapter) Completion(*gin.Context) (err error)          { return }
func (BaseAdapter) Generation(*gin.Context) (err error)          { return }
func (BaseAdapter) Embedding(*gin.Context) (err error)           { return }
func (BaseAdapter) ToolChoice(*gin.Context) (ok bool, err error) { return }
func (BaseAdapter) HandleMessages(ctx *gin.Context, completion model.Completion) (messages []model.Keyv[interface{}], err error) {
	messages = completion.Messages
	return
}
