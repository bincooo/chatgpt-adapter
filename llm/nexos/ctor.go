package nexos

import (
	"time"

	"github.com/xllm-go/g"
	"github.com/xllm-go/g/model"
	"github.com/xllm-go/g/stream"
)

const (
	comparisons = "3b528c86-9015-4286-8b27-2cc18a8541df"
)

var (
	Sdk      = g.Sdk()
	modelMap = map[string]string{
		"claude-opus-4.6": "c394b7a0-edf2-4580-b515-05c246587b49",
	}
)

func init() {
	Sdk.OnInitialized(func() {
		models := stream.Map(stream.OfMap(modelMap),
			func(pair stream.Pair[string, string]) string {
				return "nexos/" + pair.Val1
			}).ToSlice()

		Sdk.Support(models...).
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

				if err != nil {
					return err
				}

				bodies := waitChannel(ctx, response)
				return ctx.Writer(model.CreateResponse(bodies, unix))
			})
	})
}
