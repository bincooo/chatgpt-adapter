package middle

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/http"
	"time"
)

func ResponseWithE(ctx *gin.Context, err error) {
	ctx.JSON(http.StatusUnauthorized, gin.H{
		"error": map[string]string{
			"message": err.Error(),
		},
	})
}

func ResponseWithV(ctx *gin.Context, error string) {
	ctx.JSON(http.StatusUnauthorized, gin.H{
		"error": map[string]string{
			"message": error,
		},
	})
}

func ResponseWithSSE(ctx *gin.Context, model, content string, created int64) {
	w := ctx.Writer
	if w.Header().Get("Content-Type") == "" {
		ctx.Writer.Header().Set("Content-Type", "text/event-stream")
		ctx.Writer.Header().Set("Cache-Control", "no-cache")
		ctx.Writer.Header().Set("Connection", "keep-alive")
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
		Object:  "chat.completion",
		Choices: []gpt.ChatCompletionResponseChoice{
			{
				Index: 0,
				Delta: &struct {
					Role      string                   `json:"role"`
					Content   string                   `json:"content"`
					ToolCalls []map[string]interface{} `json:"tool_calls"`
				}{"assistant", content, nil},
				FinishReason: finishReason,
			},
		},
	}

	marshal, _ := json.Marshal(response)
	fmt.Fprintf(w, "data: %s\n\n", marshal)
	w.Flush()

	if done {
		fmt.Fprintf(w, "data: [DONE]")
		w.Flush()
	}
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
				}{
					Role: "assistant",
					ToolCalls: []map[string]interface{}{
						{
							"index": 0,
							"type":  "function",
							"id":    "call_" + RandomString(5),
							"function": map[string]string{
								"name": name,
							},
						},
					},
				},
			},
		},
	}

	marshal, _ := json.Marshal(response)
	fmt.Fprintf(w, "data: %s\n\n", marshal)
	w.Flush()
	time.Sleep(100 * time.Millisecond)

	response = gpt.ChatCompletionResponse{
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
				}{
					Role: "assistant",
					ToolCalls: []map[string]interface{}{
						{
							"index": 0,
							"function": map[string]string{
								"arguments": args,
							},
						},
					},
				},
			},
		},
	}
	marshal, _ = json.Marshal(response)
	fmt.Fprintf(w, "data: %s\n\n", marshal)
	w.Flush()
	time.Sleep(100 * time.Millisecond)

	response = gpt.ChatCompletionResponse{
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
				}{},
				FinishReason: "tool_calls",
			},
		},
	}
	marshal, _ = json.Marshal(response)
	fmt.Fprintf(w, "data: %s\n\n", marshal)
	w.Flush()
	time.Sleep(100 * time.Millisecond)

	fmt.Fprintf(w, "data: [DONE]")
	w.Flush()
}

func RandomString(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
