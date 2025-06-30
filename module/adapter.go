package module

import (
	"adapter/module/fiber/model"
	"github.com/gofiber/fiber/v2"
)

type RelayType byte

const (
	RELAY_TYPE_COMPLETIONS RelayType = iota
	RELAY_TYPE_EMBEDDINGS
	RELAY_TYPE_GENERATIONS
)

type Adapter interface {
	// 判定函数
	Condition(rt RelayType, ctx *fiber.Ctx) bool
	// 上下文对话
	Completions(ctx *fiber.Ctx) error
	// 向量查询
	Embeddings(ctx *fiber.Ctx) error
	// 文生图
	Generates(ctx *fiber.Ctx) error
	// 模型列表
	Models() []model.ModelEntity
	// 工具选择
	ToolChoice(ctx *fiber.Ctx) (bool, error)
}

type BasicAdapter struct {
}

func (BasicAdapter) Completions(*fiber.Ctx) error {
	return nil
}

func (BasicAdapter) Embeddings(*fiber.Ctx) error {
	return nil
}

func (BasicAdapter) Generates(*fiber.Ctx) error {
	return nil
}

func (BasicAdapter) ToolChoice(*fiber.Ctx) (ok bool, err error) { return }

func (BasicAdapter) HandleMessages(*fiber.Ctx) (messages []model.CompletionMessageEntity, err error) {
	// messages = completion.Messages
	return
}
