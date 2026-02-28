package arena

import (
	"bypass/llm/arena/headless"
	"io"
	"time"

	"github.com/xllm-go/g"
	"github.com/xllm-go/g/model"
)

var (
	Sdk       = g.Sdk()
	simulator *headless.Simulator
	models    = []string{
		"Max",
		"claude-opus-4-6-thinking",
		"claude-opus-4-6",
		"gemini-3-pro",
		"gpt-5.2-chat-latest",
		"gemini-3-flash",
		"grok-4.1-thinking",
		"claude-opus-4-5-20251101-thinking-32k",
		"claude-opus-4-5-20251101",
		"grok-4.1",
		"claude-sonnet-4-6",
		"gpt-5.1-high",
		"glm-5",
		"ernie-5.0-0110",
		"claude-sonnet-4-5-20250929-thinking-32k",
		"claude-sonnet-4-5-20250929",
		"gemini-2.5-pro",
		"ernie-5.0-preview-1203",
		"claude-opus-4-1-20250805-thinking-16k",
		"claude-opus-4-1-20250805",
		"gpt-5.2-high",
		"glm-4.7",
		"gpt-5.1",
		"gpt-5.2",
		"qwen3-max-preview",
		"kimi-k2.5-instant",
		"gpt-5-high",
		"o3-2025-04-16",
		"grok-4-1-fast-reasoning",
		"kimi-k2-thinking-turbo",
		"gpt-5-chat",
		"qwen3-max-2025-09-23",
		"claude-opus-4-20250514-thinking-16k",
		"qwen3-235b-a22b-instruct-2507",
		"grok-4-fast-chat",
		"deepseek-v3.2-thinking",
		"deepseek-v3.2",
		"kimi-k2-0905-preview",
		"kimi-k2-0711-preview",
		"mistral-large-3",
		"qwen3-vl-235b-a22b-instruct",
		"gpt-4.1-2025-04-14",
		"claude-opus-4-20250514",
		"mistral-medium-2508",
		"gemini-2.5-flash",
		"grok-4-0709",
		"claude-haiku-4-5-20251001",
		"grok-4-fast-reasoning",
		"qwen3-235b-a22b-no-thinking",
		"qwen3-next-80b-a3b-instruct",
		"minimax-m2.5",
		"longcat-flash-chat",
		"claude-sonnet-4-20250514-thinking-32k",
		"qwen3-235b-a22b-thinking-2507",
		"qwen3-vl-235b-a22b-thinking",
		"hunyuan-vision-1.5-thinking",
		"o4-mini-2025-04-16",
		"mimo-v2-flash",
		"mimo-v2-flash (thinking)",
		"gpt-5-mini-high",
		"step-3.5-flash",
		"claude-sonnet-4-20250514",
		"claude-3-7-sonnet-20250219-thinking-32k",
		"hunyuan-t1-20250711",
		"qwen3-coder-480b-a35b-instruct",
		"minimax-m2.1-preview",
		"mistral-medium-2505",
		"qwen3-30b-a3b-instruct-2507",
		"gpt-4.1-mini-2025-04-14",
		"gemini-2.5-flash-lite-preview-09-2025-no-thinking",
		"trinity-large",
		"qwen3-235b-a22b",
		"claude-3-5-sonnet-20241022",
		"claude-3-7-sonnet-20250219",
		"qwen3-next-80b-a3b-thinking",
		"minimax-m1",
		"amazon-nova-experimental-chat-11-10",
		"gemma-3-27b-it",
		"grok-3-mini-high",
		"gemini-2.0-flash-001",
		"grok-3-mini-beta",
		"intellect-3",
		"mistral-small-2506",
		"gpt-oss-120b",
		"command-a-03-2025",
		"o3-mini",
		"minimax-m2",
		"ling-flash-2.0",
		"step-3",
		"gpt-5-nano-high",
		"nova-2-lite",
		"qwq-32b",
		"olmo-3.1-32b-instruct",
		"molmo-2-8b",
		"qwen3-30b-a3b",
		"ring-flash-2.0",
		"llama-3.3-70b-instruct",
		"gemma-3n-e4b-it",
		"gpt-oss-20b",
		"nvidia-nemotron-3-nano-30b-a3b-bf16",
		"mercury",
		"olmo-3-32b-think",
		"mistral-small-3.1-24b-instruct-2503",
		"ibm-granite-h-small",
		"olmo-3.1-32b-think",
		"ling-2.5-1t",
		"ring-2.5-1t",
		"seed-1.8",
		"dola-seed-2.0-preview-vision",
		"grok-4-1-fast-non-reasoning",
		"qwen3.5-27b",
		"amazon.nova-pro-v1:0",
		"qwen3.5-35b-a3b",
		"qwen3.5-122b-a10b",
		"qwen3.5-397b-a17b",
		"amazon-nova-experimental-chat-12-10",
		"grok-4.20-beta1",
		"gemini-3.1-pro-preview",
		"gpt-5-high-new-system-prompt",
		"qwen3-vl-8b-thinking",
		"qwen3.5-flash",
		"qwen3-vl-8b-instruct",
		"glm-4.7-flash",
		"gemini-3-flash (thinking-minimal)",
		"kimi-k2.5-thinking",
		"dola-seed-2.0-preview-text",
		"qwen3-max-2025-09-26",
		"ernie-5.0-preview-1220",
		"qwen3-omni-flash",
		"qwen-vl-max-2025-08-13",
		"minimax-m2-preview",
		"qwen3-max-thinking",
	}
)

func init() {
	Sdk.OnInitialized(func() {
		proxied := Sdk.Env().GetString("server.proxied")
		simulator = headless.NewSimulator(proxied)
		Sdk.OnExited(simulator.Kill)

		slice := make([]string, len(models))
		for _, i := range models {
			slice = append(slice, "arena/"+i)
		}

		Sdk.Support(slice...).
			Relay(func(ctx *model.Ctx) (err error) {
				completion := model.JustValue[string, *model.Completion](ctx.Record, "completion")
				unix := time.Now().Unix()

				response, err := fetch(ctx)
				if err != nil {
					return err
				}

				if completion.Stream {
					ctx.StreamWriter(func(w func(interface{}) error) {
						chunkChan := createChannel(ctx, response)
						for {
							bodies, ok := <-chunkChan
							if !ok {
								err = w(io.EOF)
								break
							}

							err = w(model.CreateStreamResponse(bodies, unix))
							if err != nil {
								break
							}
						}
					})
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
