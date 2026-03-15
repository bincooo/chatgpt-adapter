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
		"Claude-Haiku-4.5":        "cca533ac-85fa-420d-bb56-692206442322",
		"Claude-Opus-4.5":         "d68c919b-74cc-4239-bcb7-99627df19098",
		"Claude-Opus-4.6":         "c394b7a0-edf2-4580-b515-05c246587b49",
		"Claude-Sonnet-4.5":       "66ecfe83-a49d-43bd-a61c-f4ab279c01e9",
		"Claude-Sonnet-4.6":       "56ba1cab-4a87-4d18-86fd-375b3c9b7c04",
		"Gemini-2.5-Flash":        "3527aee0-f13a-42c8-a10b-268a56cf52a2",
		"Gemini-2.5-Pro":          "8327b415-ce03-4d88-88a0-807edcdc4060",
		"Gemini-3-Flash-Preview":  "bb31cbea-ae45-459f-b190-fef6aa2fd6e0",
		"Gemini-3-Pro-Preview":    "7eb0d148-da72-43ba-aae9-f5f3dd902fb7",
		"Gemini-3.1-Pro-Preview":  "a31fbab9-545e-4cc9-b739-f3bb0d914ae2",
		"Imagen-4":                "0619695a-6124-4c04-adbc-ab5155fe9f0a",
		"Mistral-Large-3":         "da9d8850-8828-49ba-9232-4022ef962a9c",
		"Mistral-Medium-3":        "b1bc1216-9eb3-4ee9-b752-0f2fe53f3d50",
		"Mistral-Medium-3.1":      "f8cfc531-74bd-4fe2-a4b4-39942c94e114",
		"GPT-5":                   "d28423d2-843b-4181-93b0-b4c7b69ef376",
		"GPT-5.1":                 "73124259-b1cc-4034-99bc-a39bda499783",
		"GPT-5.2":                 "90dd544f-3186-4b91-84bf-0060a0c5ec9c",
		"Grok-4-Fast":             "863a48ed-27ea-4b11-b057-3c72f27f51c3",
		"Grok-4-Fast-Reasoning":   "496b1c4b-2c41-4dbd-8636-13f1cfe4c7a1",
		"Grok-4.1-Fast":           "b915876c-5b45-4007-acb2-1aab62f472ca",
		"Grok-4.1-Fast-Reasoning": "5b458eb1-c4d6-410f-a7f8-ab0b1737f9a3",
		"Grok-Code-Fast":          "4861e71e-0ec9-4ffe-b3c2-042df7ea52e8",
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

				channel := createChannel(ctx, response)
				return ctx.Writer(model.WaitChannelResponse(channel, unix))
			})
	})
}
