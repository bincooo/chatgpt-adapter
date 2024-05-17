package middle

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

var (
	stop      = "stop"
	toolCalls = "tool_calls"

	Delta = &struct {
		Role      string                  `json:"role"`
		Content   string                  `json:"content"`
		ToolCalls []pkg.Keyv[interface{}] `json:"tool_calls"`
	}{}
)

func MessageValidator(ctx *gin.Context) bool {
	completion := common.GetGinCompletion(ctx)
	messageL := len(completion.Messages)
	if messageL == 0 {
		ErrResponse(ctx, -1, "[] is too short - 'messages'")
		return false
	}

	condition := func(expr string) string {
		switch expr {
		case "user", "system", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	for index := 0; index < messageL; index++ {
		message := completion.Messages[index]
		role := condition(message.GetString("role"))
		if role == "" {
			str := fmt.Sprintf("'%s' is not in ['system', 'assistant', 'user', 'function'] - 'messages.[%d].role'", message["role"], index)
			ErrResponse(ctx, -1, str)
			return false
		}
	}
	return true
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
func ErrResponse(ctx *gin.Context, code int, err interface{}) {
	logrus.Errorf("response error: %v", err)
	if code == -1 {
		code = http.StatusInternalServerError
	}

	if str, ok := err.(string); ok {
		ctx.JSON(code, gin.H{
			"error": map[string]string{
				"message": str,
			},
		})
		return
	}

	if e, ok := err.(error); ok {
		ctx.JSON(code, gin.H{
			"error": map[string]string{
				"message": e.Error(),
			},
		})
		return
	}

	ctx.JSON(code, gin.H{
		"error": map[string]string{
			"message": fmt.Sprintf("%v", err),
		},
	})
}

func Response(ctx *gin.Context, model, content string) {
	created := time.Now().Unix()
	ctx.JSON(http.StatusOK, pkg.ChatResponse{
		Model:   model,
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion",
		Choices: []pkg.ChatChoice{
			{
				Index: 0,
				Message: &struct {
					Role      string                  `json:"role"`
					Content   string                  `json:"content"`
					ToolCalls []pkg.Keyv[interface{}] `json:"tool_calls"`
				}{"assistant", content, nil},
				FinishReason: &stop,
			},
		},
	})
}

func SSEResponse(ctx *gin.Context, model, content string, usage map[string]int, created int64) {
	setSSEHeader(ctx)

	done := false
	finishReason := ""

	if content == "[DONE]" {
		done = true
		content = ""
		finishReason = "stop"
	}

	response := pkg.ChatResponse{
		Model:   model,
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion.chunk",
		Choices: []pkg.ChatChoice{
			{
				Index: 0,
				Delta: &struct {
					Role      string                  `json:"role"`
					Content   string                  `json:"content"`
					ToolCalls []pkg.Keyv[interface{}] `json:"tool_calls"`
				}{"assistant", content, nil},
			},
		},
		Usage: usage,
	}

	if finishReason != "" {
		response.Choices[0].FinishReason = &finishReason
	}

	event(ctx.Writer, response)

	if done {
		time.Sleep(100 * time.Millisecond)
		event(ctx.Writer, "[DONE]")
	}
}

func ToolCallResponse(ctx *gin.Context, model, name, args string) {
	created := time.Now().Unix()
	ctx.JSON(http.StatusOK, pkg.ChatResponse{
		Model:   model,
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion",
		Choices: []pkg.ChatChoice{
			{
				Index: 0,
				Message: &struct {
					Role      string                  `json:"role"`
					Content   string                  `json:"content"`
					ToolCalls []pkg.Keyv[interface{}] `json:"tool_calls"`
				}{
					Role: "assistant",
					ToolCalls: []pkg.Keyv[interface{}]{
						{
							"id":   "call" + common.RandStr(5),
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

func SSEToolCallResponse(ctx *gin.Context, model, name, args string, created int64) {
	setSSEHeader(ctx)
	response := pkg.ChatResponse{
		Model:   model,
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion.chunk",
		Choices: []pkg.ChatChoice{
			{
				Index: 0,
				Delta: &struct {
					Role      string                  `json:"role"`
					Content   string                  `json:"content"`
					ToolCalls []pkg.Keyv[interface{}] `json:"tool_calls"`
				}{
					Role:      "assistant",
					ToolCalls: make([]pkg.Keyv[interface{}], 1),
				},
			},
		},
	}

	toolCall := make(map[string]interface{})
	toolCall["index"] = 0
	toolCall["type"] = "function"
	toolCall["id"] = "call" + common.RandStr(5)
	toolCall["function"] = map[string]string{"name": name}
	response.Choices[0].Delta.ToolCalls[0] = toolCall

	event(ctx.Writer, response)
	time.Sleep(100 * time.Millisecond)

	delete(toolCall, "id")
	delete(toolCall, "type")
	toolCall["function"] = map[string]string{"arguments": args}
	response.Choices[0].Delta.ToolCalls[0] = toolCall

	event(ctx.Writer, response)
	time.Sleep(100 * time.Millisecond)

	response.Choices[0].FinishReason = &toolCalls
	response.Choices[0].Delta = Delta

	event(ctx.Writer, response)
	time.Sleep(100 * time.Millisecond)
	event(ctx.Writer, "[DONE]")
}

func setSSEHeader(ctx *gin.Context) {
	h := ctx.Writer.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "text/event-stream")
		h.Set("Transfer-Encoding", "chunked")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
	}
}

func event(w gin.ResponseWriter, data interface{}) {
	str, ok := data.(string)
	if ok {
		_, err := fmt.Fprintf(w, "data: %s\n\n", str)
		if err != nil {
			logrus.Error(err)
			return
		}

		w.Flush()
		return
	}

	marshal, err := json.Marshal(data)
	if err != nil {
		logrus.Error(err)
		return
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", marshal)
	if err != nil {
		logrus.Error(err)
		return
	}
	w.Flush()
}
