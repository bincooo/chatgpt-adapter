package fiber

import (
	"adapter/module"
	"adapter/module/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

var busInterfaces = make([]module.Adapter, 0)

func init() {
	// TODO -
}

// 初始化fiber api
func Initialized(addr string) {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app.Get("/", index)

	app.Post("v1/chat/completions", completions)
	app.Post("v1/object/completions", completions)
	app.Post("proxies/v1/chat/completions", completions)

	app.Post("/v1/embeddings", embeddings)
	app.Post("proxies/v1/embeddings", embeddings)

	app.Post("v1/images/generations", generations)
	app.Post("v1/object/generations", generations)
	app.Post("proxies/v1/images/generations", generations)

	err := app.Listen(addr)
	if err != nil {
		panic(err)
	}
}

func index(ctx *fiber.Ctx) error {
	return common.JustError(
		ctx.WriteString("<div style='color:green'>success ~</div>"),
	)
}

func completions(ctx *fiber.Ctx) (err error) {
	// TODO -
	for _, instance := range busInterfaces {
		if instance.Condition(module.RELAY_TYPE_COMPLETIONS, ctx) {
			err = instance.Completions(ctx)
			break
		}
	}
	return
}

func embeddings(ctx *fiber.Ctx) (err error) {
	// TODO -
	for _, instance := range busInterfaces {
		if instance.Condition(module.RELAY_TYPE_EMBEDDINGS, ctx) {
			err = instance.Embeddings(ctx)
			break
		}
	}
	return
}

func generations(ctx *fiber.Ctx) (err error) {
	// TODO -
	for _, instance := range busInterfaces {
		if instance.Condition(module.RELAY_TYPE_GENERATIONS, ctx) {
			err = instance.Generates(ctx)
			break
		}
	}
	return
}
