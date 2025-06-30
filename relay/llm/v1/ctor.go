package v1

import (
	"adapter/module"
	"adapter/module/fiber/model"
	"github.com/gofiber/fiber/v2"
)

var (
	_ module.Adapter = (*Ada)(nil)
)

type Ada struct {
	module.BasicAdapter
}

func New() *Ada { return &Ada{} }

func (api *Ada) Condition(rt module.RelayType, ctx *fiber.Ctx) bool {
	panic("implement me")
}

func (api *Ada) Models() []model.ModelEntity {
	panic("implement me")
}
