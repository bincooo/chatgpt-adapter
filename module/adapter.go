package module

import "github.com/gofiber/fiber/v2"

type RelayType byte

const (
	RELAY_TYPE_COMPLETIONS RelayType = iota
	RELAY_TYPE_EMBEDDINGS
	RELAY_TYPE_GENERATIONS
)

type Adapter interface {
	Condition(rt RelayType, ctx *fiber.Ctx) bool
	Completions(ctx *fiber.Ctx) error
	Embeddings(ctx *fiber.Ctx) error
	Generates(ctx *fiber.Ctx) error
}

type BasicAdapter struct {
}

func Completions(ctx *fiber.Ctx) error {
	return nil
}

func Embeddings(ctx *fiber.Ctx) error {
	return nil
}

func Generates(ctx *fiber.Ctx) error {
	return nil
}
