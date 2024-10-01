package you

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"errors"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func waitMessage(ch chan string, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-ch
		if !ok {
			break
		}

		if strings.HasPrefix(message, "error:") {
			return "", logger.WarpError(errors.New(message[6:]))
		}

		if strings.HasPrefix(message, "limits:") {
			continue
		}

		if len(message) > 0 {
			content += message
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, cancel chan error, ch chan string, sse bool) (content string) {
	var (
		created = time.Now().Unix()
		tokens  = ctx.GetInt(ginTokens)
	)

	logger.Info("waitResponse ...")
	for {
		select {
		case err := <-cancel:
			if err != nil {
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, err)
				}
				return
			}
			goto label
		default:
			message, ok := <-ch
			if !ok {
				raw := common.ExecMatchers(matchers, "", true)
				if raw != "" && sse {
					response.SSEResponse(ctx, Model, raw, created)
				}
				content += raw
				goto label
			}

			if strings.HasPrefix(message, "error:") {
				logger.Error(message[6:])
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, message[6:])
				}
				return
			}

			if strings.HasPrefix(message, "limits:") {
				continue
			}

			var raw = message
			logger.Debug("----- raw -----")
			logger.Debug(raw)
			raw = common.ExecMatchers(matchers, raw, false)
			if len(raw) == 0 {
				continue
			}

			if sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
		}
	}

label:
	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}

	return
}

func waitMessageResponse(ctx *gin.Context, ch chan string, matchers []common.Matcher, cancel chan error) (content string) {
	logger.Info("waitResponse ...")
	for {
		select {
		case <-cancel:
			logger.Error("context deadline exceeded")
			if response.NotSSEHeader(ctx) {
				response.Error(ctx, -1, "context deadline exceeded")
			}
			return
		default:
			message, ok := <-ch
			if !ok {
				raw := common.ExecMatchers(matchers, "", true)
				if raw != "" {
					response.Event(ctx, "content_block_delta", map[string]interface{}{
						"index": 0,
						"type":  "content_block_delta",
						"delta": map[string]interface{}{
							"type": "text_delta", "text": raw,
						},
					})
				}
				content += raw
				goto label
			}

			if strings.HasPrefix(message, "error:") {
				logger.Error(message[6:])
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, message[6:])
				}
				return
			}

			if strings.HasPrefix(message, "limits:") {
				continue
			}

			logger.Debug("----- raw -----")
			logger.Debug(message)
			raw := common.ExecMatchers(matchers, message, false)
			if len(raw) == 0 {
				continue
			}

			response.Event(ctx, "content_block_delta", map[string]interface{}{
				"index": 0,
				"type":  "content_block_delta",
				"delta": map[string]interface{}{
					"type": "text_delta", "text": raw,
				},
			})
			content += raw
		}
	}

label:
	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	response.Event(ctx, "message_delta", map[string]interface{}{
		"type":  "message_delta",
		"usage": map[string]int{"output_tokens": common.CalcTokens(content)},
		"delta": map[string]interface{}{
			"stop_reason":   "end_turn",
			"stop_sequence": nil,
		},
	})
	time.Sleep(time.Second)
	response.Event(ctx, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})
	return
}

func mergeMessages(ctx *gin.Context, completion pkg.ChatCompletion) (fileMessage string, text string, tokens int, err error) {
	text = notice
	var (
		messages     = completion.Messages
		toolMessages = common.FindToolMessages(&completion)
	)

	if messages, err = common.HandleMessages(completion, nil); err != nil {
		return
	}
	messages = append(messages, toolMessages...)

	var (
		pos      = 0
		contents []string
	)
	messageL := len(messages)
	for {
		if pos > messageL-1 {
			break
		}

		message := messages[pos]
		role, end := common.ConvertRole(ctx, message.GetString("role"))
		contents = append(contents, role+message.GetString("content")+end)
		pos++
	}

	fileMessage = strings.Join(contents, "")
	if strings.HasSuffix(fileMessage, "<|end|>\n\n") {
		fileMessage = fileMessage[:len(fileMessage)-9]
	}
	tokens += common.CalcTokens(fileMessage)
	return
}
