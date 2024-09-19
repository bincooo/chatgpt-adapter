package gemini

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
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

const ginTokens = "__tokens__"

func waitResponse(ctx *gin.Context, matchers []common.Matcher, partialResponse *http.Response, sse bool) (content string) {
	defer partialResponse.Body.Close()

	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)
	completion := common.GetGinCompletion(ctx)
	toolId := common.GetGinToolValue(ctx).GetString("id")
	toolId = plugin.NameWithTools(toolId, completion.Tools)

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
			raw := common.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, MODEL, raw, created)
			}
			content += raw
			break
		}

		if err != nil {
			logger.Error(err)
			if response.NotSSEHeader(ctx) {
				logger.Error(err)
				response.Error(ctx, -1, err)
			}
			return
		}

		if len(original) == 0 {
			continue
		}

		logrus.Tracef("--------- ORIGINAL MESSAGE ---------")
		logrus.Tracef("%s", original)

		if bytes.Contains(original, []byte(`"error":`)) {
			err = fmt.Errorf("%s", original)
			logger.Error(err)
			if response.NotSSEHeader(ctx) {
				logger.Error(err)
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

		if len(c.Candidates) == 0 {
			continue
		}

		cond := c.Candidates[0]
		if cond.Content.Role != "model" {
			original = nil
			continue
		}

		if len(cond.Content.Parts) == 0 {
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
		raw = common.ExecMatchers(matchers, raw.(string), false)
		if len(raw.(string)) == 0 {
			continue
		}

		if toolId != "-1" {
			functionCall = map[string]interface{}{
				"name": toolId,
				"args": map[string]interface{}{},
			}
			break
		}

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

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, MODEL, content)
	} else {
		response.SSEResponse(ctx, MODEL, "[DONE]", created)
	}
	return
}

func waitMessage(partialResponse *http.Response, cancel func(str string) bool) (string, error) {
	defer partialResponse.Body.Close()
	reader := bufio.NewReader(partialResponse.Body)
	var original []byte
	var block = []byte("data: ")
	content := ""

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
			return "", err
		}

		if len(original) == 0 {
			continue
		}

		if bytes.Contains(original, []byte(`"error":`)) {
			return "", fmt.Errorf("%s", original)
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

		if len(c.Candidates) == 0 {
			continue
		}

		cond := c.Candidates[0]
		if cond.Content.Role != "model" {
			original = nil
			continue
		}

		if len(cond.Content.Parts) == 0 {
			continue
		}

		raw, ok := cond.Content.Parts[0]["text"]
		if !ok {
			original = nil
			continue
		}

		original = nil
		if len(raw.(string)) == 0 {
			continue
		}

		if cancel != nil && cancel(raw.(string)) {
			return content + raw.(string), nil
		}
		content += raw.(string)
	}
	return content, nil
}

func mergeMessages(messages []pkg.Keyv[interface{}]) (newMessages []map[string]interface{}, tokens int, err error) {
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

	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (result []map[string]interface{}, err error) {
		role := opts.Message["role"]
		tokens += common.CalcTokens(opts.Message["content"])
		if condition(role) == condition(opts.Next) {
			// cache buffer
			opts.Buffer.WriteString(opts.Message["content"])
			return
		}

		defer opts.Buffer.Reset()
		opts.Buffer.WriteString(fmt.Sprintf(opts.Message["content"]))

		// 工具执行结果消息
		if role == "tool" {
			var args interface{}
			if err = json.Unmarshal([]byte(opts.Message["content"]), &args); err != nil {
				err = logger.WarpError(err)
				return
			}

			result = append(result, map[string]interface{}{
				"role": condition("user"),
				"parts": []interface{}{
					map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name":     opts.Message["name"],
							"response": args,
						},
					},
				},
			})
			return
		}

		// 工具参数消息
		if _, ok := opts.Message["toolCalls"]; ok && role == "assistant" {
			var args interface{}
			if err = json.Unmarshal([]byte(opts.Message["content"]), &args); err != nil {
				err = logger.WarpError(err)
				return
			}

			result = append(result, map[string]interface{}{
				"role": condition("assistant"),
				"parts": []interface{}{
					map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": opts.Message["name"],
							"args": args,
						},
					},
				},
			})
			return
		}

		// 复合消息
		if _, ok := opts.Message["multi"]; ok && role == "user" {
			message := opts.Initial()
			values := message.GetSlice("content")
			if len(values) == 0 {
				return
			}

			var multi []interface{}
			for _, value := range values {
				var keyv pkg.Keyv[interface{}]
				keyv, ok = value.(map[string]interface{})
				if !ok {
					continue
				}

				if keyv.Is("type", "text") {
					multi = append(multi, map[string]interface{}{
						"text": keyv.GetString("text"),
					})
				}

				if keyv.Is("type", "image_url") {
					o := keyv.GetKeyv("image_url")
					mime, data, e := common.LoadImageMeta(o.GetString("url"))
					if e != nil {
						err = logger.WarpError(e)
						return
					}
					multi = append(multi, map[string]interface{}{
						"inlineData": map[string]interface{}{
							"mimeType": mime,
							"data":     data,
						},
					})
				}
			}

			if len(multi) == 0 {
				return
			}

			result = append(result, map[string]interface{}{
				"role":  condition("user"),
				"parts": multi,
			})
			return
		}

		if role == "system" {
			result = append(result, map[string]interface{}{
				"role": "user",
				"parts": []interface{}{
					map[string]string{
						"text": opts.Buffer.String(),
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
			return
		}

		result = []map[string]interface{}{
			{
				"role": condition(role),
				"parts": []interface{}{
					map[string]string{
						"text": opts.Buffer.String(),
					},
				},
			},
		}
		return
	}

	newMessages, err = common.TextMessageCombiner(messages, iterator)
	if err != nil {
		err = logger.WarpError(err)
	}
	return
}
