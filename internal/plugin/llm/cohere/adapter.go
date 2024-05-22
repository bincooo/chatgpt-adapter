package cohere

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	coh "github.com/bincooo/cohere-api"
	"github.com/gin-gonic/gin"
)

var (
	Adapter = API{}
	Model   = "cohere"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case coh.COMMAND,
		coh.COMMAND_R,
		coh.COMMAND_LIGHT,
		coh.COMMAND_LIGHT_NIGHTLY,
		coh.COMMAND_NIGHTLY,
		coh.COMMAND_R_PLUS:
		return true
	default:
		return false
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "command",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-r",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-light",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-light-nightly",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-nightly",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		}, {
			Id:      "command-r-plus",
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		notebook   = ctx.GetBool("notebook")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	var (
		system    string
		message   string
		pMessages []coh.Message
		chat      coh.Chat
		//tools     = convertTools(completion)
	)

	// 官方的文档toolCall描述十分模糊，简测功能不佳，改回提示词实现
	if /*notebook &&*/ plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	// TODO - 官方Go库出了，后续修改
	if notebook {
		message = mergeMessages(completion.Messages)
		ctx.Set(ginTokens, common.CalcTokens(message))
		chat = coh.New(cookie, completion.Temperature, completion.Model, false)
		chat.Proxies(proxies)
		chat.TopK(completion.TopK)
		chat.MaxTokens(completion.MaxTokens)
		chat.StopSequences([]string{
			"user:",
			"assistant:",
			"system:",
		})
	} else {
		// chat模式已实现toolCall
		var tokens = 0
		pMessages, system, message, tokens = mergeChatMessages(completion.Messages)
		ctx.Set(ginTokens, tokens)
		chat = coh.New(cookie, completion.Temperature, completion.Model, true)
		chat.Proxies(proxies)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), pMessages, system, message, nil)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	waitResponse(ctx, matchers, chatResponse, completion.Stream)
}

func convertTools(completion pkg.ChatCompletion) (tools []coh.Tool) {
	if len(completion.Tools) == 0 {
		return
	}

	condition := func(str string) string {
		switch str {
		case "string":
			return "str"
		case "boolean":
			return "bool"
		case "number":
			return str
		default:
			return "object"
		}
	}

	contains := func(slice []interface{}, str string) bool {
		for _, v := range slice {
			if v == str {
				return true
			}
		}
		return false
	}

	for pos := range completion.Tools {
		t := completion.Tools[pos]
		if !t.Is("type", "function") {
			continue
		}

		fn := t.GetKeyv("function")
		params := make(map[string]interface{})
		if fn.Has("parameters") {
			keyv := fn.GetKeyv("parameters")
			properties := keyv.GetKeyv("properties")
			required := keyv.GetSlice("required")
			for k, v := range properties {
				value := v.(map[string]interface{})
				value["required"] = contains(required, k)
				value["type"] = condition(value["type"].(string))
				params[k] = value
			}
		}

		tools = append(tools, coh.Tool{
			Name:        fn.GetString("name"),
			Description: fn.GetString("description"),
			Param:       params,
		})
	}
	return
}
