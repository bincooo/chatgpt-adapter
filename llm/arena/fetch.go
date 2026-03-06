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

	message, err := model.JinjaMessage(headless.JinjaTemplate, completion)
	if err != nil {
		logger.Sugar().Error(err)
		return
	}

	incognitoTab, err := simulator.Launch(ctx.Context(), ctx.Token)
	if err != nil {
		logger.Sugar().Error(err)
		return
	}

	Sdk.OnPanic(func(err interface{}) {
		incognitoTab.Close()
	})

	return incognitoTab.Relay(completion.Model[6:], message)
}
