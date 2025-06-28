package fiber

import (
	"adapter/module/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

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
	return
}

func embeddings(ctx *fiber.Ctx) (err error) {
	// TODO -
	return
}

func generations(ctx *fiber.Ctx) (err error) {
	// TODO -
	return
}
