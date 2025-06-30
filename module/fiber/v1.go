package fiber

import (
	"adapter/module"
	"adapter/module/common"
	"adapter/module/fiber/model"
	"adapter/relay/llm/v1"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

var (
	store    = session.New()
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

func GetSession(ctx *fiber.Ctx) (*session.Session, error) {
	return store.Get(ctx)
}

// 初始化fiber api
func Initialized(addr string) {
	app := fiber.New()
	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: "*",
	// 	AllowHeaders: "Origin, Content-Type, Accept",
	// }))

	// 初始化session
	app.Use(func(ctx *fiber.Ctx) error {
		sessionStore, err := store.Get(ctx)
		if err != nil {
			return err
		}

		err = sessionStore.Save()
		if err != nil {
			return err
		}

		return ctx.Next()
	})

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

	sessionStore, err := GetSession(ctx)
	if err != nil {
		return
	}

	sessionStore.Set("completion", completion)
	for _, adapter := range adapters {
		if adapter.Condition(module.RELAY_TYPE_COMPLETIONS, ctx) {
			err = adapter.Completions(ctx)
			break
		}
	}
	return
}

func embeddings(ctx *fiber.Ctx) (err error) {
	embedding := new(model.EmbeddingEntity)
	if err = ctx.BodyParser(embedding); err != nil {
		return
	}

	sessionStore, err := GetSession(ctx)
	if err != nil {
		return
	}

	sessionStore.Set("embedding", embedding)
	for _, adapter := range adapters {
		if adapter.Condition(module.RELAY_TYPE_EMBEDDINGS, ctx) {
			err = adapter.Embeddings(ctx)
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

	sessionStore, err := GetSession(ctx)
	if err != nil {
		return
	}

	sessionStore.Set("generation", generation)
	for _, adapter := range adapters {
		if adapter.Condition(module.RELAY_TYPE_GENERATIONS, ctx) {
			err = adapter.Generates(ctx)
			break
		}
	}
	return
}
