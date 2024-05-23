package gemini

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	com "github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	goole "github.com/bincooo/goole15"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func waitResponse(ctx *gin.Context, matchers []com.Matcher, partialResponse *http.Response, sse bool) {
	content := ""
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)

	reader := bufio.NewReader(partialResponse.Body)
	var original []byte
	var block = []byte("data: ")
	var functionCall interface{}

	for {
		line, hm, err := reader.ReadLine()
		original = append(original, line...)
		if hm {
			continue
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			logger.Error(err)
			if response.NotSSEHeader(ctx) {
				response.Error(ctx, -1, err)
			}
			return
		}

		if len(original) == 0 {
			continue
		}

		if bytes.Contains(original, []byte(`"error":`)) {
			err = fmt.Errorf("%s", original)
			logger.Error(err)
			if response.NotSSEHeader(ctx) {
				response.Error(ctx, -1, err)
			}
			return
		}

		if !bytes.HasPrefix(original, block) {
			continue
		}

		var c candidatesResponse
		original = bytes.TrimPrefix(original, block)
		if err = json.Unmarshal(original, &c); err != nil {
			logger.Error(err)
			continue
		}

		cond := c.Candidates[0]
		if cond.Content.Role != "model" {
			original = nil
			continue
		}

		if fc, ok := cond.Content.Parts[0]["functionCall"]; ok {
			functionCall = fc
			original = nil
			continue
		}

		raw, ok := cond.Content.Parts[0]["text"]
		if !ok {
			original = nil
			continue
		}

		logger.Debug("----- raw -----")
		logger.Debug(raw)

		original = nil
		raw = com.ExecMatchers(matchers, raw.(string))

		if sse {
			response.SSEResponse(ctx, MODEL, raw.(string), created)
		}
		content += raw.(string)

	}

	if functionCall != nil {
		fc := functionCall.(map[string]interface{})
		args, _ := json.Marshal(fc["args"])
		if sse {
			response.SSEToolCallResponse(ctx, MODEL, fc["name"].(string), string(args), created)
		} else {
			response.ToolCallResponse(ctx, MODEL, fc["name"].(string), string(args))
		}
		return
	}

	ctx.Set(vars.GinCompletionUsage, com.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, MODEL, content)
	} else {
		response.SSEResponse(ctx, MODEL, "[DONE]", created)
	}
}

func waitResponse15(ctx *gin.Context, matchers []com.Matcher, ch chan string, sse bool) {
	content := ""
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)

	for {
		tex, ok := <-ch
		if !ok {
			break
		}

		if strings.HasPrefix(tex, "error: ") {
			err := strings.TrimPrefix(tex, "error: ")
			logger.Error(err)
			if response.NotSSEHeader(ctx) {
				response.Error(ctx, -1, err)
			}
			return
		}

		if strings.HasPrefix(tex, "text: ") {
			raw := strings.TrimPrefix(tex, "text: ")

			logger.Debug("----- raw -----")
			logger.Debug(raw)

			raw = com.ExecMatchers(matchers, raw)
			if sse {
				response.SSEResponse(ctx, MODEL+"-1.5", raw, created)
			}
			content += raw
		}
	}

	ctx.Set(vars.GinCompletionUsage, com.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, MODEL+"-1.5", content)
	} else {
		response.SSEResponse(ctx, MODEL+"-1.5", "[DONE]", created)
	}
}

func mergeMessages(messages []pkg.Keyv[interface{}]) (newMessages []map[string]interface{}, tokens int) {
	// role类型转换
	condition := func(expr string) string {
		switch expr {
		case "function", "tool", "end":
			return expr
		case "assistant":
			return "model"
		default:
			return "user"
		}
	}

	newMessages = com.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []map[string]interface{} {
		role := message["role"]
		tokens += com.CalcTokens(message["content"])
		if condition(role) == condition(next) {
			// cache buffer
			buffer.WriteString(message["content"])
			return nil
		}

		defer buffer.Reset()
		buffer.WriteString(fmt.Sprintf(message["content"]))
		var result []map[string]interface{}

		if role == "tool" || role == "function" {
			var args interface{}
			if err := json.Unmarshal([]byte(message["content"]), &args); err != nil {
				logger.Error(err)
				return nil
			}

			result = append(result, map[string]interface{}{
				"role": "user",
				"parts": []interface{}{
					map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name":     message["name"],
							"response": args,
						},
					},
				},
			})
			return result
		}

		if toolCalls, ok := message["tool_calls"]; ok && role == "assistant" && toolCalls == "yes" {
			var args interface{}
			if err := json.Unmarshal([]byte(message["content"]), &args); err != nil {
				logger.Error(err)
				return nil
			}

			result = append(result, map[string]interface{}{
				"role": "assistant",
				"parts": []interface{}{
					map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": message["name"],
							"args": args,
						},
					},
				},
			})
			return result
		}

		if role == "system" {
			result = append(result, map[string]interface{}{
				"role": "user",
				"parts": []interface{}{
					map[string]string{
						"text": buffer.String(),
					},
				},
			})
			result = append(result, map[string]interface{}{
				"role": "model",
				"parts": []interface{}{
					map[string]string{
						"text": "ok ~",
					},
				},
			})
			return result
		}

		return []map[string]interface{}{
			{
				"role": condition(role),
				"parts": []interface{}{
					map[string]string{
						"text": buffer.String(),
					},
				},
			},
		}
	})

	return
}

func mergeMessages15(messages []pkg.Keyv[interface{}]) (newMessages []goole.Message, tokens int) {
	condition := func(expr string) string {
		switch expr {
		case "user", "system", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	newMessages = com.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []goole.Message {
		role := message["role"]
		tokens += com.CalcTokens(message["content"])
		if condition(role) == condition(next) {
			// cache buffer
			if role == "function" {
				buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], message["content"]))
				return nil
			}
			buffer.WriteString(message["content"])
			return nil
		}

		defer buffer.Reset()
		buffer.WriteString(fmt.Sprintf(message["content"]))
		return []goole.Message{
			{
				Role:    role,
				Content: buffer.String(),
			},
		}
	})

	if newMessages[0].Role != "user" {
		newMessages = append([]goole.Message{
			{
				Role:    "user",
				Content: "hi ~",
			},
		}, newMessages...)
	}

	return
}
