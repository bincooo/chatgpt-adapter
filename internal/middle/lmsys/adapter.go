package lmsys

import (
	"errors"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"strings"
)

var (
	Adapter = API{}
	Model   = "lmsys"
)

type API struct {
	middle.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	return strings.HasPrefix(model, "lmsys/")
}

func (API) Models() []middle.Model {
	/*
		// lmsys 模型导出代码
		const lis = $0.querySelectorAll('li')
		let result = ''
		for (let index = 0, len = lis.length; index < len; index ++) {
		    result += `{
						"id":       "lmsys/${lis[index].getAttribute('aria-label')}",
						"object":   "model",
						"created":  1686935002,
						"owned_by": "lmsys-adapter",
					}, `
		}
		console.log(result)
	*/
	return []middle.Model{
		{
			Id:      "lmsys/claude-3-haiku-20240307",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/claude-3-sonnet-20240229",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/claude-3-opus-20240229",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/claude-2.1",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/gpt-4o-2024-05-13",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/gpt-4-turbo-2024-04-09",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/im-also-a-good-gpt2-chatbot",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/im-a-good-gpt2-chatbot",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/llama-3-70b-instruct",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/llama-3-8b-instruct",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/gemini-1.5-pro-api-preview",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/reka-core-20240501",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/qwen-max-0428",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/qwen1.5-110b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/snowflake-arctic-instruct",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/phi-3-mini-128k-instruct",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/mixtral-8x22b-instruct-v0.1",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/gpt-3.5-turbo-0125",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/reka-flash",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/reka-flash-online",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/command-r-plus",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/command-r",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/gemma-1.1-7b-it",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/gemma-1.1-2b-it",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/mixtral-8x7b-instruct-v0.1",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/mistral-large-2402",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/mistral-medium",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/qwen1.5-72b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/qwen1.5-32b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/qwen1.5-14b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/qwen1.5-7b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/qwen1.5-4b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/zephyr-orpo-141b-A35b-v0.1",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/dbrx-instruct",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/llama-2-70b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/llama-2-13b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/llama-2-7b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/olmo-7b-instruct",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/vicuna-13b",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/yi-34b-chat",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/codellama-70b-instruct",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		}, {
			Id:      "lmsys/openhermes-2.5-mistral-7b",
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	completion.Model = completion.Model[6:]
	if common.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, proxies, completion) {
			return
		}
	}

	newMessages := mergeMessages(completion.Messages)
	ctx.Set("tokens", common.CalcTokens(newMessages))
	retry := 3
label:
	ch, err := fetch(ctx, proxies, newMessages, options{
		model:       completion.Model,
		temperature: completion.Temperature,
		topP:        completion.TopP,
		maxTokens:   completion.MaxTokens,
	})
	if err != nil {
		if retry > 0 {
			retry--
			goto label
		}

		middle.ErrResponse(ctx, -1, err)
		return
	}

	cancel, matchers := joinMatchers(ctx, matchers)
	waitResponse(ctx, matchers, ch, cancel, completion.Stream)
}

func joinMatchers(ctx *gin.Context, matchers []pkg.Matcher) (chan error, []pkg.Matcher) {
	// 自定义标记块中断
	cancel, matcher := pkg.NewCancelMather(ctx)
	matchers = append(matchers, matcher)

	// 违反内容中断并返回错误1
	matchers = append(matchers, &pkg.SymbolMatcher{
		Find: "I did not actually provide",
		H: func(index int, content string) (state int, result string) {
			cancel <- errors.New("SECURITY POLICY INTERCEPTION")
			return vars.MatMatched, ""
		},
	})

	// 违反内容中断并返回错误2
	matchers = append(matchers, &pkg.SymbolMatcher{
		Find: "I apologize",
		H: func(index int, content string) (state int, result string) {
			cancel <- errors.New("SECURITY POLICY INTERCEPTION")
			return vars.MatMatched, ""
		},
	})
	return cancel, matchers
}
