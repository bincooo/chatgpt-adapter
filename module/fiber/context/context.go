package context

import (
	"adapter/module/fiber/model"
	"github.com/gofiber/fiber/v2"
	"strings"
)

type Ctx struct {
	ctx *fiber.Ctx
	model.Record[string, any]

	Token string
}

func New(ctx *fiber.Ctx) *Ctx {
	return &Ctx{
		ctx:    ctx,
		Record: make(model.Record[string, any]),

		Token: token(ctx),
	}
}

func token(ctx *fiber.Ctx) (token string) {
	token = ctx.Get("X-Api-Key")
	if token == "" {
		token = strings.TrimPrefix(ctx.Get("Authorization"), "Bearer ")
	}
	return
}
