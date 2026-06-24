package arena

import (
	"bypass/llm/arena/headless"
	"strconv"
	"time"

	"github.com/xllm-go/g"
	"github.com/xllm-go/g/model"
)

var (
	Sdk       = g.Sdk()
	simulator *headless.Simulator
	models    = []string{
		"Max",
		"gemini-3-flash",
		"gpt-5.2-chat-latest",
		"glm-5.1",
		"qwen3.5-397b-a17b",
		"claude-sonnet-4-5-20250929",
		"gemini-3.1-pro-preview",
		"minimax-m3",
		"gemini-2.5-pro",
		"claude-haiku-4-5-20251001",
		"glm-5v-turbo",
		"grok-4.20-beta-0309-reasoning",
		"gpt-5.2-high-no-system-prompt-text",
		"gpt-5.5-instant-2026-05-28",
		"gpt-5.1",
		"gpt-5.2-no-system-prompt-text",
		"grok-4.20-multi-agent-beta-0309",
		"claude-sonnet-4-6-vertex",
		"kiteki",
		"glm-5",
		"claude-sonnet-4-5-20250929-thinking-32k",
		"gpt-5.1-high",
		"gpt-5.3-chat-latest",
		"mimo-v2-pro",
		"gpt-5.4-mini-high",
		"glm-4.7-text-fireworks",
		"qwen3-max-preview-v2",
		"gpt-5-high",
		"gemini-3.1-flash-lite-preview",
		"mimo-v2-omni",
		"kimi-k2.5-instant-20260302",
		"o3-2025-04-16",
		"kimi-k2-thinking-turbo",
		"gpt-5-chat",
		"qwen3-max-2025-09-23",
		"qwen3-235b-a22b-instruct-2507",
		"kimi-k2-0905-preview",
		"kimi-k2-0711-preview",
		"qwen3.5-122b-a10b",
		"deep-octo",
		"jaguar",
		"qwen3-vl-235b-a22b-instruct",
		"gpt-4.1-2025-04-14",
		"gemini-2.5-flash",
		"mistral-medium-2508",
		"qwen3.5-27b",
		"qwen3-235b-a22b-no-thinking",
		"gpt-5.4-nano-high",
		"qwen3-next-80b-a3b-instruct",
		"longcat-flash-chat",
		"qwen3-235b-a22b-thinking-2507",
		"claude-sonnet-4-20250514-thinking-32k",
		"qwen3.5-flash",
		"hunyuan-vision-1.5-thinking",
		"qwen3.5-35b-a3b",
		"qwen3-vl-235b-a22b-thinking",
		"step-3.5-flash-openrouter",
		"mimo-v2-flash-thinking-v2",
		"mimo-v2-flash",
		"minimax-m2.5",
		"gpt-5-mini-high",
		"o4-mini-2025-04-16",
		"claude-sonnet-4-20250514",
		"qwen3-coder-480b-a35b-instruct",
		"mistral-medium-2505",
		"minimax-m2.1-preview",
		"qwen3-30b-a3b-instruct-2507",
		"gpt-4.1-mini-2025-04-14",
		"trinity-large",
		"qwen3-235b-a22b",
		"qwen3-next-80b-a3b-thinking",
		"trinity-large-thinking",
		"gemma-3-27b-it",
		"minimax-m1",
		"gemini-2.0-flash-001",
		"mistral-small-2506",
		"intellect-3",
		"gpt-oss-120b",
		"o3-mini",
		"mercury-2",
		"ling-flash-2.0",
		"minimax-m2",
		"global.amazon.nova-2-lite-v1:0",
		"gpt-5-nano-high",
		"qwq-32b",
		"olmo-3.1-32b-instruct",
		"qwen3-30b-a3b",
		"ring-flash-2.0",
		"gemma-3n-e4b-it",
		"gpt-oss-20b",
		"december-chatbot",
		"granite-4.1-8b",
		"mercury",
		"mistral-small-3.1-24b-instruct-2503",
		"ibm-granite-h-small",
		"olmo-3.1-32b-think-v2",
		"ring-2.5-1t-20260217",
		"ling-2.5-1t",
		"dola-seed-2.0-preview-vision",
		"march26-chatbot1-public",
		"dola-seed-2.0-preview-text",
		"qwen3.6-27b",
		"pteronura",
		"kizen-alpha",
		"may26-chatbot4-public",
		"significant-otter",
		"mimo-v2.5-pro",
		"mistral-small-2603",
		"ernie-5.0-preview-1220",
		"kimi-k2.5-20260327",
		"gpt-5-high-new-system-prompt",
		"deepseek-v4-pro-thinking-public",
		"qwen3-vl-8b-thinking",
		"mizar-v2-85jb",
		"qwen3.6-plus-text",
		"glm-5.2",
		"may-alpha-0k1k",
		"mistral-medium-3.5",
		"deepseek-v4-flash",
		"kimi-k2.6-20260420",
		"qwen3.7-plus",
		"gemini-3-flash-thinking-minimal-fixed-20251224",
		"qwen-vl-max-2025-08-13",
		"qwen3-omni-flash",
		"qwen3-vl-8b-instruct",
		"grok-4.3",
		"gemini-3.5-flash",
		"mimo-v2.5",
		"amazon.nova-pro-v1:0",
		"deepseek-v4-flash-thinking",
		"minimax-m2-preview",
		"deepseek-v4-pro-public",
		"melyora-9qr6",
		"qwen3-max-thinking",
		"qwen3-max-2025-09-26",
	}
)

func init() {
	Sdk.OnInitialized(func() {
		environ := Sdk.Env()
		proxied := environ.GetString("server.proxied")
		bin := environ.GetString("headless.bin")
		maxIdle := environ.GetInt("headless.maxIdle")
		nopeCHAToken := environ.GetString("headless.nopeCHAToken")
		str := environ.GetString("headless.enabled")

		if maxIdle == 0 {
			maxIdle = 10
		}

		less := true
		if b, err := strconv.ParseBool(str); err == nil {
			less = b
		}

		simulator = headless.NewSimulator(proxied, bin, less,
			headless.WithMax(maxIdle),
			headless.WithNopeCHAToken(nopeCHAToken),
		)
		Sdk.OnExited(simulator.Kill)

		slice := make([]string, 0)
		for _, i := range models {
			slice = append(slice, "arena/"+i)
		}

		Sdk.Support(slice...).
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
