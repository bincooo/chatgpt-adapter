package fiber

import (
	"adapter/module"
	"adapter/module/common"
	"adapter/module/fiber/context"
	"adapter/module/fiber/model"
	"adapter/module/llm"
	"adapter/module/logger"
	"adapter/relay/llm/v1"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var (
	adapters = make([]module.Adapter, 0)
)

func init() {
	adapters = append(adapters,
		v1.New(),
	)
}

func ModelEach(yield func(index int, model model.ModelEntity)) {
	idx := 0
	for _, adapter := range adapters {
		for _, mod := range adapter.Models() {
			yield(idx, mod)
			idx++
		}
	}
}

// 初始化fiber api
func Initialized(addr string) {
	app := fiber.New()
	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: "*",
	// 	AllowHeaders: "Origin, Content-Type, Accept",
	// }))

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(ctx *fiber.Ctx, err interface{}) {
			logger.Sugar().Errorf("panic: %v", err)
		},
	}))
	app.Use(fiberzap.New(fiberzap.Config{
		Logger: logger.Logger(),
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
	ctx.Set("content-type", "text/html")
	return common.JustError(
		ctx.WriteString("<div style='color:green'>success ~</div>"),
	)
}

func completions(ctx *fiber.Ctx) (err error) {
	completion := new(model.CompletionEntity)
	if err = ctx.BodyParser(completion); err != nil {
		return
	}

	ctx.Get("")
	c := context.New(ctx)
	c.Put("completion", completion)
	for _, adapter := range adapters {
		if !adapter.Condition(module.RELAY_TYPE_COMPLETIONS, c, completion.Model) {
			continue
		}

		completion.Messages, err = adapter.HandleMessages(c)
		if err != nil {
			return
		}

		var choice = false
		if llm.WillToolExecute(c) {
			choice, err = adapter.ToolExecuted(c)
			if err != nil {
				return
			}
		}

		if !choice {
			err = adapter.Completions(c)
		}
		break
	}
	return
}

func embeddings(ctx *fiber.Ctx) (err error) {
	embedding := new(model.EmbeddingEntity)
	if err = ctx.BodyParser(embedding); err != nil {
		return
	}

	c := context.New(ctx)
	c.Put("embedding", embedding)
	for _, adapter := range adapters {
		if adapter.Condition(module.RELAY_TYPE_EMBEDDINGS, c, embedding.Model) {
			err = adapter.Embeddings(c)
			break
		}
	}
	return
}

func generations(ctx *fiber.Ctx) (err error) {
	generation := new(model.GenerationEntity)
	if err = ctx.BodyParser(generation); err != nil {
		return
	}

	c := context.New(ctx)
	c.Put("generation", generation)
	for _, adapter := range adapters {
		if adapter.Condition(module.RELAY_TYPE_GENERATIONS, c, generation.Model) {
			err = adapter.Generates(c)
			break
		}
	}
	return
}
