package middle

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"time"
)

func ResponseWithE(ctx *gin.Context, err error) {
	logrus.Error("response error: ", err)
	ctx.JSON(http.StatusUnauthorized, gin.H{
		"error": map[string]string{
			"message": err.Error(),
		},
	})
}

func ResponseWithV(ctx *gin.Context, error string) {
	logrus.Errorf("response error: %s", error)
	ctx.JSON(http.StatusUnauthorized, gin.H{
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
				FinishReason: "stop",
			},
		},
	})
}

func ResponseWithSSE(ctx *gin.Context, model, content string, created int64) {
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
							"id":   "call_" + RandString(5),
							"type": "function",
							"function": map[string]string{
								"name":      name,
								"arguments": args,
							},
						},
					},
				},
				FinishReason: "stop",
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
	toolCall["id"] = "call_" + RandString(5)
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

	response.Choices[index].FinishReason = "tool_calls"
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

func RandString(n int) string {
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	bytes := make([]rune, n)
	for i := range bytes {
		bytes[i] = runes[rand.Intn(len(runes))]
	}
	return string(bytes)
}

func Contains[T comparable](slice []T, t T) bool {
	if slice == nil {
		return false
	}
	for _, item := range slice {
		if item == t {
			return true
		}
	}
	return false
}
