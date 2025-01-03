package lmsys

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

var (
	Model = "lmsys"

	/*
		// lmsys 模型导出代码
		const lis = $0.querySelectorAll('li')
		let result = ''
		for (let index = 0, len = lis.length; index < len; index ++) {
			result += `"${lis[index].getAttribute('aria-label')}",\n`
		}
		console.log(`[${result}]`)
	*/
	modelSlice = []string{
		"chatgpt-4o-latest-20241120",
		"gemini-exp-1121",
		"gemini-exp-1114",
		"chatgpt-4o-latest-20240903",
		"gpt-4o-mini-2024-07-18",
		"gpt-4o-2024-08-06",
		"gpt-4o-2024-05-13",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"grok-2-2024-08-13",
		"grok-2-mini-2024-08-13",
		"gemini-1.5-pro-002",
		"gemini-1.5-flash-002",
		"gemini-1.5-flash-8b-001",
		"gemini-1.5-pro-001",
		"gemini-1.5-flash-001",
		"llama-3.1-nemotron-70b-instruct",
		"llama-3.1-nemotron-51b-instruct",
		"llama-3.2-vision-90b-instruct",
		"llama-3.2-vision-11b-instruct",
		"llama-3.1-405b-instruct-bf16",
		"llama-3.1-405b-instruct-fp8",
		"llama-3.1-70b-instruct",
		"llama-3.1-8b-instruct",
		"llama-3.2-3b-instruct",
		"llama-3.2-1b-instruct",
		"hunyuan-standard-256k",
		"mistral-large-2411",
		"pixtral-large-2411",
		"mistral-large-2407",
		"yi-lightning",
		"yi-vision",
		"glm-4-plus",
		"molmo-72b-0924",
		"molmo-7b-d-0924",
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
		"athene-v2-chat",
		"qwen2.5-coder-32b-instruct",
		"qwen2.5-72b-instruct",
		"qwen-max-0919",
		"qwen-plus-0828",
		"qwen-vl-max-0809",
		"gpt-3.5-turbo-0125",
		"phi-3-mini-4k-instruct-june-2024",
		"reka-core-20240904",
		"reka-flash-20240904",
		"c4ai-aya-expanse-32b",
		"command-r-plus-08-2024",
		"command-r-08-2024",
		"codestral-2405",
		"mixtral-8x22b-instruct-v0.1",
		"f1-mini-preview",
		"mixtral-8x7b-instruct-v0.1",
		"pixtral-12b-2409",
		"ministral-8b-2410",
		"internvl2-26b",
		"qwen2-vl-7b-instruct",
		"internvl2-4b",
	}
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	token := ctx.GetString("token")
	if len(model) <= 6 || model[:6] != Model+"/" {
		return
	}

	slice := api.env.GetStringSlice("lmsys.model")
	for _, mod := range append(slice, modelSlice...) {
		if model[6:] != mod {
			continue
		}

		password := api.env.GetString("server.password")
		if password != "" && password != token {
			err = response.UnauthorizedError
			return
		}

		ok = true
	}
	return
}

func (api *api) Models() (result []model.Model) {
	slice := api.env.GetStringSlice("lmsys.model")
	for _, mod := range append(slice, modelSlice...) {
		result = append(result, model.Model{
			Id:      "lmsys/" + mod,
			Object:  "model",
			Created: 1686935002,
			By:      "lmsys-adapter",
		})
	}
	return
}

func (api *api) ToolChoice(ctx *gin.Context) (ok bool, err error) {
	var (
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	if toolChoice(ctx, api.env, proxied, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	completion.Model = completion.Model[6:]
	newMessages, err := mergeMessages(ctx, completion)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}
	ctx.Set(ginTokens, response.CalcTokens(newMessages))
	ch, err := fetch(ctx.Request.Context(), api.env, proxied, newMessages,
		options{
			model:       completion.Model,
			temperature: completion.Temperature,
			topP:        completion.TopP,
			maxTokens:   completion.MaxTokens,
		})
	if err != nil {
		logger.Error(err)
		return
	}

	content := waitResponse(ctx, ch, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
	return
}
