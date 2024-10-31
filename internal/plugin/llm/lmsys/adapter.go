package lmsys

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"errors"
	"github.com/gin-gonic/gin"
	"strings"
)

var (
	Adapter = API{}
	Model   = "lmsys"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	return strings.HasPrefix(model, "lmsys/")
}

func (API) Models() (result []plugin.Model) {
	/*
		// lmsys 模型导出代码
		const lis = $0.querySelectorAll('li')
		let result = ''
		for (let index = 0, len = lis.length; index < len; index ++) {
			result += `"${lis[index].getAttribute('aria-label')}",\n`
		}
		console.log(`[${result}]`)
	*/
	slice := []string{
		"chatgpt-4o-latest-20240903",
		"gpt-4o-mini-2024-07-18",
		"gpt-4o-2024-08-06",
		"gpt-4o-2024-05-13",
		"grok-2-2024-08-13",
		"grok-2-mini-2024-08-13",
		"gemini-1.5-pro-exp-0827",
		"gemini-1.5-flash-exp-0827",
		"gemini-1.5-flash-8b-exp-0827",
		"gemini-1.5-pro-api-0514",
		"gemini-1.5-flash-api-0514",
		"claude-3-5-sonnet-20240620",
		"llama-3.1-405b-instruct-bf16",
		"llama-3.1-405b-instruct-fp8",
		"llama-3.1-70b-instruct",
		"llama-3.1-8b-instruct",
		"mistral-large-2407",
		"im-also-a-good-gpt2-chatbot",
		"im-a-good-gpt2-chatbot",
		"jamba-1.5-large",
		"jamba-1.5-mini",
		"gemma-2-27b-it",
		"gemma-2-9b-it",
		"gemma-2-2b-it",
		"eureka-chatbot",
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
		"deepseek-v2.5",
		"nemotron-4-340b",
		"llama-3-70b-instruct",
		"llama-3-8b-instruct",
		"athene-70b-0725",
		"qwen2.5-72b-instruct",
		"qwen2-72b-instruct",
		"qwen-max-0428",
		"qwen-plus-0828",
		"qwen-vl-max-0809",
		"gpt-3.5-turbo-0125",
		"yi-large-preview",
		"yi-large",
		"yi-vision",
		"yi-1.5-34b-chat",
		"phi-3-mini-4k-instruct-june-2024",
		"reka-core-20240904",
		"reka-core-20240722",
		"reka-flash-20240904",
		"reka-flash-20240722",
		"command-r-plus",
		"command-r-plus-08-2024",
		"command-r",
		"command-r-08-2024",
		"codestral-2405",
		"mixtral-8x22b-instruct-v0.1",
		"mixtral-8x7b-instruct-v0.1",
		"mistral-large-2402",
		"mistral-medium",
		"qwen1.5-110b-chat",
		"qwen1.5-72b-chat",
		"glm-4-0520",
		"glm-4-0116",
		"dbrx-instruct",
		"internvl2-26b",
		"internlm2_5-20b-chat",
		"qwen2-vl-7b-instruct",
		"phi-3.5-vision-instruct",
		"llava-onevision-qwen2-72b-ov",
		"pixtral-12b-2409",
		"internvl2-4b",
	}

	for _, model := range slice {
		result = append(result, plugin.Model{
			Id:      "lmsys/" + model,
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		})
	}

	return
}

func (API) HandleMessages(ctx *gin.Context) (messages []pkg.Keyv[interface{}], err error) {
	var (
		completion   = common.GetGinCompletion(ctx)
		toolMessages = common.FindToolMessages(&completion)
	)

	if messages, err = common.HandleMessages(completion, nil); err != nil {
		return
	}
	messages = append(messages, toolMessages...)
	return
}

func (API) Completion(ctx *gin.Context) {
	var (
		token      = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
		echo       = ctx.GetBool(vars.GinEcho)
	)

	completion.Model = completion.Model[6:]
	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, proxies, token, completion) {
			return
		}
	}

	newMessages, err := mergeMessages(ctx, completion)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	ctx.Set(ginTokens, common.CalcTokens(newMessages))
	if echo {
		response.Echo(ctx, completion.Model, newMessages, completion.Stream)
		return
	}

	retry := 3
label:
	ch, err := fetch(common.GetGinContext(ctx), proxies, token, newMessages, options{
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

		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	cancel, matchers := joinMatchers(ctx, matchers)
	content := waitResponse(ctx, matchers, ch, cancel, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func joinMatchers(ctx *gin.Context, matchers []common.Matcher) (chan error, []common.Matcher) {
	// 自定义标记块中断
	cancel, matcher := common.NewCancelMatcher(ctx)
	matchers = append(matchers, matcher...)

	// 违反内容中断并返回错误1
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "I did not actually provide",
		H: func(index int, content string) (state int, _, result string) {
			cancel <- errors.New("SECURITY POLICY INTERCEPTION")
			return vars.MatMatched, "", ""
		},
	})

	// 违反内容中断并返回错误2
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "I apologize",
		H: func(index int, content string) (state int, _, result string) {
			cancel <- errors.New("SECURITY POLICY INTERCEPTION")
			return vars.MatMatched, "", ""
		},
	})
	return cancel, matchers
}
