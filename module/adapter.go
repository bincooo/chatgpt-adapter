package module

import (
	"adapter/module/fiber/context"
	"adapter/module/fiber/model"
	"errors"
)

type RelayType byte

const (
	RELAY_TYPE_COMPLETIONS RelayType = iota
	RELAY_TYPE_EMBEDDINGS
	RELAY_TYPE_GENERATIONS
)

type Adapter interface {
	// 判定函数
	Condition(rt RelayType, ctx *context.Ctx, model string) bool
	// 上下文对话
	Completions(ctx *context.Ctx) error
	// 向量查询
	Embeddings(ctx *context.Ctx) error
	// 文生图
	Generates(ctx *context.Ctx) error
	// 模型列表
	Models() []model.ModelEntity
	// 工具选择
	ToolExecuted(ctx *context.Ctx) (bool, error)
	// 上下文处理
	HandleMessages(ctx *context.Ctx) ([]model.CompletionMessageEntity, error)
}

type BasicAdapter struct {
}

func (BasicAdapter) Completions(*context.Ctx) error {
	return nil
}

func (BasicAdapter) Embeddings(*context.Ctx) error {
	return nil
}

func (BasicAdapter) Generates(*context.Ctx) error {
	return nil
}

func (BasicAdapter) ToolExecuted(*context.Ctx) (ok bool, err error) { return }

func (BasicAdapter) HandleMessages(ctx *context.Ctx) (messages []model.CompletionMessageEntity, err error) {
	completion, ok := model.GetValue[string, *model.CompletionEntity](ctx.Record, "completion")
	if !ok {
		err = errors.New("completion not found")
		return
	}
	messages = completion.Messages
	return
}
