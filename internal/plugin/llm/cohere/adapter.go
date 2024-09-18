package cohere

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"fmt"
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
		coh.COMMAND_R_PLUS,
		coh.COMMAND_R_202408,
		coh.COMMAND_R_PLUS_202408:
		return true
	default:
		return false
	}
}

func (API) Models() (result []plugin.Model) {
	slice := []string{
		coh.COMMAND,
		coh.COMMAND_R,
		coh.COMMAND_LIGHT,
		coh.COMMAND_LIGHT_NIGHTLY,
		coh.COMMAND_NIGHTLY,
		coh.COMMAND_R_PLUS,
		coh.COMMAND_R_202408,
		coh.COMMAND_R_PLUS_202408,
	}
	for _, model := range slice {
		result = append(result, plugin.Model{
			Id:      model,
			Object:  "model",
			Created: 1686935002,
			By:      "cohere-adapter",
		})
	}
	return
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
		//toolObject = coh.ToolObject{
		//	Tools:   convertTools(completion),
		//	Results: convertToolResults(completion),
		//}

		echo = ctx.GetBool(vars.GinEcho)
	)

	// 官方的文档toolCall描述十分模糊，简测功能不佳，改回提示词实现
	if /*notebook &&*/ plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	// TODO - 官方Go库出了，后续修改
	if notebook {
		//toolObject = coh.ToolObject{}
		message = mergeMessages(ctx, completion.Messages)
		ctx.Set(ginTokens, common.CalcTokens(message))
		if echo {
			response.Echo(ctx, completion.Model, message, completion.Stream)
			return
		}

		chat = coh.New(cookie, completion.Temperature, completion.Model, false)
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
		if echo {
			bytes, _ := json.MarshalIndent(pMessages, "", "  ")
			response.Echo(ctx, completion.Model, fmt.Sprintf("SYSTEM\n%s\n\n\n-------PREVIOUS MESSAGES:\n%s\n\n\n------\nCURR QUESTION:\n%s", system, bytes, message), completion.Stream)
			return
		}

		chat = coh.New(cookie, completion.Temperature, completion.Model, true)
		chat.TopK(completion.TopK)
		chat.MaxTokens(completion.MaxTokens)
		chat.StopSequences(completion.StopSequences)
		if completion.Model == coh.COMMAND_R_202408 || completion.Model == coh.COMMAND_R_PLUS_202408 {
			if safety := pkg.Config.GetString("cohere.safety"); safety != "" {
				chat.Safety(safety)
			}
		}
	}

	chat.Proxies(proxies)
	chat.Client(plugin.HTTPClient)
	chatResponse, err := chat.Reply(common.GetGinContext(ctx), pMessages, system, message, coh.ToolObject{})
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	content := waitResponse(ctx, matchers, chatResponse, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func convertToolResults(completion pkg.ChatCompletion) (toolResults []coh.ToolResult) {
	find := func(name string) map[string]interface{} {
		for pos := range completion.Messages {
			message := completion.Messages[pos]
			if !message.Is("role", "assistant") || !message.Has("tool_calls") {
				continue
			}

			toolCalls := message.GetSlice("tool_calls")
			if len(toolCalls) == 0 {
				continue
			}

			var toolCall pkg.Keyv[interface{}] = toolCalls[0].(map[string]interface{})
			if !toolCall.Has("function") {
				continue
			}

			var args interface{}
			fn := toolCall.GetKeyv("function")
			if !fn.Is("name", name) {
				continue
			}

			if err := json.Unmarshal([]byte(fn.GetString("arguments")), &args); err != nil {
				logger.Error(err)
				continue
			}

			return map[string]interface{}{
				"name":       name,
				"parameters": args,
			}
		}
		return nil
	}

	for pos := range completion.Messages {
		message := completion.Messages[pos]
		if message.Is("role", "tool") {
			call := find(message.GetString("name"))
			if call == nil {
				continue
			}

			var output interface{}
			if err := json.Unmarshal([]byte(message.GetString("content")), &output); err != nil {
				logger.Error(err)
				continue
			}

			toolResults = append(toolResults, coh.ToolResult{
				Call: call,
				Outputs: []interface{}{
					output,
				},
			})
		}
	}
	return
}

func convertTools(completion pkg.ChatCompletion) (tools []coh.ToolCall) {
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

		tools = append(tools, coh.ToolCall{
			Name:        fn.GetString("name"),
			Description: fn.GetString("description"),
			Param:       params,
		})
	}
	return
}
