package chatgpt

import (
	"time"

	"github.com/xllm-go/g"
	"github.com/xllm-go/g/model"
)

var (
	Sdk = g.Sdk()
)

func init() {
	Sdk.OnInitialized(func() {
		Sdk.Support("chatgpt/GPT-5.3-mini").
			Relay(func(ctx *model.Ctx) (err error) {
				completion := ctx.GetCompletion()
				unix := time.Now().Unix()
				response, err := fetch(ctx)
				if err != nil {
					return err
				}

				if completion.Stream {
					channel := createChannel(ctx, response)
					ctx.StreamWriter(channel, unix)
					return
				}

				channel := createChannel(ctx, response)
				return ctx.Writer(model.WaitChannelResponse(channel, unix))
			})
	})
}
