package arena

import (
	"bypass/jinja"
	"io"

	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

func fetch(ctx *model.Ctx) (reader io.Reader, err error) {
	completion := ctx.GetCompletion()
	message, err := model.JinjaMessage(jinja.DefaultTemplate, completion)
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
