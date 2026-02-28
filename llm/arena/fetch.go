package arena

import (
	"io"

	"bypass/llm/arena/headless"

	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

func fetch(ctx *model.Ctx) (reader io.Reader, err error) {
	var (
		completion = model.JustValue[string, *model.Completion](ctx.Record, "completion")
	)

	mod := completion.Model[6:]
	// TODO -
	content, err := model.JinjaMessage(headless.JinjaTemplate, completion)
	if err != nil {
		logger.Sugar().Error(err)
		return
	}

	id := simulator.Launch(ctx.Ctx().Context(), ctx.Token)
	Sdk.OnPanic(func(err interface{}) {
		simulator.Close(id)
	})

	reader, err = simulator.Relay(id, mod, content)
	return
}
