package middle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

var (
	ContentCanceled = errors.New("request canceled")
	stop            = "stop"
	toolCalls       = "tool_calls"
)

func IsCanceled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ContentCanceled
	default:
		return nil
	}
}

func ResponseWithE(ctx *gin.Context, code int, err error) {
	ResponseWithV(ctx, code, err.Error())
}

// one-api 重试机制
//
//	read to https://github.com/songquanpeng/one-api/blob/5e81e19bc81e88d5df15a04f6a6268886127e002/controller/relay.go#L99
//	code 429 http.StatusTooManyRequests
//	code 5xx
//
// one-api 自动关闭管道
//
//	https://github.com/songquanpeng/one-api/blob/5e81e19bc81e88d5df15a04f6a6268886127e002/controller/relay.go#L118
//	code 401 http.StatusUnauthorized
//	err.Type ...
func ResponseWithV(ctx *gin.Context, code int, error string) {
	logrus.Errorf("response error: %s", error)
	if code == -1 {
		code = http.StatusInternalServerError
	}
	ctx.JSON(code, gin.H{
		// "code": "invalid_api_key",
		"error": map[string]string{
			"message": error,
		},
	})
}

func ResponseWith(ctx *gin.Context, model, content string) {
	created := time.Now().Unix()
	ctx.JSON(http.StatusOK, gpt.ChatCompletionResponse{
		Model:   model,
		Created: created,
		Id:      "chatcmpl-completion",
		Object:  "chat.completion",
		Choices: []gpt.ChatCompletionResponseChoice{
			{
				Index: 0,
				Message: &struct {
					Role      string                   `json:"role"`
					Content   string                   `json:"content"`
					ToolCalls []map[string]interface{} `json:"tool_calls"`
				}{"assistant", content, nil},
				FinishReason: &stop,
			},
		},
	})
}

func ResponseWithSSE(ctx *gin.Context, model, content string, usage map[string]int, created int64) {
	w := ctx.Writer
	if w.Header().Get("Content-Type") == "" {
		ctx.Writer.Header().Set("Content-Type", "text/event-stream")
		ctx.Writer.Header().Set("Transfer-Encoding", "chunked")
		ctx.Writer.Header().Set("Cache-Control", "no-cache")
		ctx.Writer.Header().Set("Connection", "keep-alive")
		ctx.Writer.Header().Set("X-Accel-Buffering", "no")
	}

	done := false
	finishReason := ""

	if content == "[DONE]" {
		done = true
		content = ""
		finishReason = "stop"
	}

	response := gpt.ChatCompletionResponse{
		Model:   model,
		Created: created,
		Id:      "chatcmpl-completion",
		Object:  "chat.completion.chunk",
		Choices: []gpt.ChatCompletionResponseChoice{
			{
				Index: 0,
				Delta: &struct {
					Role      string                   `json:"role"`
					Content   string                   `json:"content"`
					ToolCalls []map[string]interface{} `json:"tool_calls"`
				}{"assistant", content, nil},
				// FinishReason: finishReason,
			},
		},
		Usage: usage,
	}

	if finishReason != "" {
		response.Choices[0].FinishReason = &finishReason
	}

	marshal, _ := json.Marshal(response)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", marshal)
	w.Flush()

	if done {
		_, _ = fmt.Fprintf(w, "data: [DONE]")
		w.Flush()
	}
}

func ResponseWithToolCalls(ctx *gin.Context, model, name, args string) {
	created := time.Now().Unix()
	ctx.JSON(http.StatusOK, gpt.ChatCompletionResponse{
		Model:   model,
		Created: created,
		Id:      "chatcmpl-completion",
		Object:  "chat.completion",
		Choices: []gpt.ChatCompletionResponseChoice{
			{
				Index: 0,
				Message: &struct {
					Role      string                   `json:"role"`
					Content   string                   `json:"content"`
					ToolCalls []map[string]interface{} `json:"tool_calls"`
				}{
					Role: "assistant",
					ToolCalls: []map[string]interface{}{
						{
							"id":   "call_" + common.RandStr(5),
							"type": "function",
							"function": map[string]string{
								"name":      name,
								"arguments": args,
							},
						},
					},
				},
				FinishReason: &stop,
			},
		},
	})
}

func ResponseWithSSEToolCalls(ctx *gin.Context, model, name, args string, created int64) {
	w := ctx.Writer
	if w.Header().Get("Content-Type") == "" {
		ctx.Writer.Header().Set("Content-Type", "text/event-stream")
		ctx.Writer.Header().Set("Transfer-Encoding", "chunked")
		ctx.Writer.Header().Set("Cache-Control", "no-cache")
		ctx.Writer.Header().Set("Connection", "keep-alive")
		ctx.Writer.Header().Set("X-Accel-Buffering", "no")
	}

	index := 0
	response := gpt.ChatCompletionResponse{
		Model:   model,
		Created: created,
		Id:      "chatcmpl-completion",
		Object:  "chat.completion.chunk",
		Choices: []gpt.ChatCompletionResponseChoice{
			{
				Index: index,
				Delta: &struct {
					Role      string                   `json:"role"`
					Content   string                   `json:"content"`
					ToolCalls []map[string]interface{} `json:"tool_calls"`
				}{
					Role:      "assistant",
					ToolCalls: make([]map[string]interface{}, 1),
				},
			},
		},
	}

	toolCall := make(map[string]interface{})
	toolCall["index"] = index
	toolCall["type"] = "function"
	toolCall["id"] = "call_" + common.RandStr(5)
	toolCall["function"] = map[string]string{"name": name}
	response.Choices[index].Delta.ToolCalls[index] = toolCall

	marshal, _ := json.Marshal(response)
	_, err := fmt.Fprintf(w, "data: %s\n\n", marshal)
	if err != nil {
		return
	}
	w.Flush()
	time.Sleep(100 * time.Millisecond)

	delete(toolCall, "id")
	delete(toolCall, "type")
	toolCall["function"] = map[string]string{"arguments": args}
	response.Choices[index].Delta.ToolCalls[index] = toolCall
	marshal, _ = json.Marshal(response)
	_, err = fmt.Fprintf(w, "data: %s\n\n", marshal)
	if err != nil {
		return
	}
	w.Flush()
	time.Sleep(100 * time.Millisecond)

	response.Choices[index].FinishReason = &toolCalls
	response.Choices[index].Delta = &struct {
		Role      string                   `json:"role"`
		Content   string                   `json:"content"`
		ToolCalls []map[string]interface{} `json:"tool_calls"`
	}{}
	marshal, _ = json.Marshal(response)
	_, err = fmt.Fprintf(w, "data: %s\n\n", marshal)
	if err != nil {
		return
	}
	w.Flush()
	time.Sleep(100 * time.Millisecond)

	_, _ = fmt.Fprintf(w, "data: [DONE]")
	w.Flush()
}
