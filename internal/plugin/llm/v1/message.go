package v1

import (
	"bufio"
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const ginTokens = "__tokens__"

func waitMessage(r *http.Response, cancel func(str string) bool) (content string, err error) {
	defer r.Body.Close()

	scanner := bufio.NewScanner(r.Body)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if eof {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for {
		if !scanner.Scan() {
			break
		}

		data := scanner.Text()
		if len(data) < 6 || data[:6] != "data: " {
			continue
		}

		data = data[6:]
		if data == "[DONE]" {
			break
		}

		var chat pkg.ChatResponse
		err = json.Unmarshal([]byte(data), &chat)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		if len(chat.Choices) == 0 {
			continue
		}

		choice := chat.Choices[0]
		if choice.Delta.Role != "" && choice.Delta.Role != "assistant" {
			continue
		}

		if choice.FinishReason != nil && *choice.FinishReason == "stop" {
			continue
		}

		raw := choice.Delta.Content
		if len(raw) == 0 {
			continue
		}

		content += raw
		if cancel != nil && cancel(content) {
			return content, nil
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, r *http.Response, sse bool) (content string) {
	defer r.Body.Close()

	logger.Info("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)
	completion := common.GetGinCompletion(ctx)
	toolId := common.GetGinToolValue(ctx).GetString("id")
	toolId = plugin.NameWithTools(toolId, completion.Tools)
	var toolCall map[string]interface{}
	htc := false

	scanner := bufio.NewScanner(r.Body)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if eof {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for {
		if !scanner.Scan() {
			break
		}

		data := scanner.Text()
		logger.Tracef("--------- ORIGINAL MESSAGE ---------")
		logger.Tracef("%s", data)

		if len(data) < 6 || data[:6] != "data: " {
			continue
		}

		data = data[6:]
		if data == "[DONE]" {
			if htc && !sse {
				toolCall["args"] = content
			}
			break
		}

		var chat pkg.ChatResponse
		err := json.Unmarshal([]byte(data), &chat)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		if len(chat.Choices) == 0 {
			continue
		}

		choice := chat.Choices[0]
		if choice.Delta.Role != "" && choice.Delta.Role != "assistant" {
			continue
		}

		if choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0 {
			htc = true
			if sse {
				response.Event(ctx, "", chat)
				continue
			}

			keyv := choice.Delta.ToolCalls[0].GetKeyv("function")
			if name := keyv.GetString("name"); name != "" {
				toolCall = map[string]interface{}{
					"name": name,
					"args": "",
				}
			}
			content += keyv.GetString("arguments")
			continue
		}

		if choice.FinishReason != nil && *choice.FinishReason == "stop" {
			if chat.Usage == nil {
				chat.Usage = common.CalcUsageTokens(content, tokens)
			}
			ctx.Set(vars.GinCompletionUsage, chat.Usage)
			if sse {
				response.Event(ctx, "", chat)
			}
			continue
		}

		raw := choice.Delta.Content
		logger.Debug("----- raw -----")
		logger.Debug(raw)

		raw = common.ExecMatchers(matchers, raw)
		if len(raw) == 0 {
			continue
		}

		if !htc && toolId != "-1" {
			toolCall = map[string]interface{}{
				"name": toolId,
				"args": "",
			}
			break
		}

		choice.Delta.Content = raw
		if sse && len(raw) > 0 {
			response.Event(ctx, "", chat)
		}
		content += raw
	}

	if toolCall != nil {
		if !sse {
			response.ToolCallResponse(ctx, Model, toolCall["name"].(string), toolCall["args"].(string))
		} else {
			response.SSEToolCallResponse(ctx, Model, toolCall["name"].(string), toolCall["args"].(string), time.Now().Unix())
		}
		return
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.Event(ctx, "", "[DONE]")
	}
	return
}
