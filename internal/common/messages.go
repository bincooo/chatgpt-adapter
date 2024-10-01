package common

import (
	"bytes"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

func ConvertRole(ctx *gin.Context, role string) (newRole, end string) {
	completion := GetGinCompletion(ctx)
	if IsClaude(ctx, "", completion.Model) {
		switch role {
		case "user":
			newRole = "\n\r\nHuman: "
		case "assistant":
			newRole = "\n\r\nAssistant: "
		}
		return
	}

	end = "<|end|>\n\n"
	if IsGPT(completion.Model) {
		switch role {
		case "user":
			newRole = "<|start|>user\n"
		case "assistant":
			newRole = "<|start|>assistant\n"
		default:
			newRole = "<|start|>system\n"
		}
		return
	}

	newRole = "<|" + role + "|>\n"
	return
}

func IsGPT(model string) bool {
	model = strings.ToLower(model)
	return strings.Contains(model, "openai") || strings.Contains(model, "gpt")
}

func FindToolMessages(completion *pkg.ChatCompletion) (toolMessages []pkg.Keyv[interface{}]) {
	for i := len(completion.Messages) - 1; i >= 0; i-- {
		message := completion.Messages[i]
		if message.Is("role", "tool") || (message.Is("role", "assistant") && message.Has("tool_calls")) {
			toolMessages = append(toolMessages, message)
		} else {
			completion.Messages = completion.Messages[:i+1]
			break
		}
	}
	return
}

func TextMessageCombiner[T any](
	messages []pkg.Keyv[interface{}],
	iterator func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) ([]T, error),
) (newMessages []T, err error) {
	// 要命，当时为什么要做消息合并
	previous := "start"
	buffer := new(bytes.Buffer)
	tempMessages := make([]map[string]string, 0)
	sources := make(map[string]pkg.Keyv[interface{}])

	cleanBuffer := func() {
		if buffer.Len() > 0 {
			tempMessages = append(tempMessages, map[string]string{
				"role":    previous,
				"content": buffer.String(),
			})
			buffer.Reset()
		}
	}

	toolCallMessages := func(message pkg.Keyv[interface{}]) {
		cleanBuffer()
		previous = message.GetString("role")
		toolCalls := message.GetSlice("tool_calls")
		if len(toolCalls) == 0 {
			return
		}

		var toolCall pkg.Keyv[interface{}] = toolCalls[0].(map[string]interface{})
		keyv := toolCall.GetKeyv("function")

		id := uuid.NewString()
		sources[id] = message
		tempMessages = append(tempMessages, map[string]string{
			"role":      previous,
			"name":      keyv.GetString("name"),
			"content":   keyv.GetString("arguments"),
			"toolCalls": "yes",
			"id":        id,
		})
	}

	toolResponses := func(message pkg.Keyv[interface{}]) {
		cleanBuffer()
		previous = message.GetString("role")
		id := uuid.NewString()
		sources[id] = message
		tempMessages = append(tempMessages, map[string]string{
			"role":    previous,
			"name":    message.GetString("name"),
			"content": message.GetString("content"),
			"tool":    "yes",
			"id":      id,
		})
	}

	multiResponses := func(message pkg.Keyv[interface{}]) {
		cleanBuffer()
		previous = message.GetString("role")
		values := message.GetSlice("content")
		if len(values) == 0 {
			return
		}

		var (
			contents []string
			keyv     pkg.Keyv[interface{}]
			ok       bool
		)

		for _, value := range values {
			keyv, ok = value.(map[string]interface{})
			if !ok {
				continue
			}
			if !keyv.Is("type", "text") {
				continue
			}
			contents = append(contents, keyv.GetString("text"))
		}

		id := uuid.NewString()
		sources[id] = message
		tempMessages = append(tempMessages, map[string]string{
			"role":    previous,
			"content": strings.Join(contents, "\n\n"),
			"multi":   "yes",
			"id":      id,
		})
	}

	for _, message := range messages {
		// toolCalls
		if message.Is("role", "assistant") && message.Has("tool_calls") {
			toolCallMessages(message)
			continue
		}

		// tool
		if message.Is("role", "tool") || message.Is("role", "function") {
			toolResponses(message)
			continue
		}

		// multi content
		if message.Is("role", "user") && !message.IsString("content") {
			multiResponses(message)
			continue
		}

		// is str content
		str := strings.TrimSpace(message.GetString("content"))
		if str == "" {
			continue
		}

		if buffer.Len() != 0 {
			buffer.WriteString("\n\n")
		}

		{ // 相同类型消息合并
			if previous == "start" {
				previous = message.GetString("role")
				buffer.WriteString(str)
				continue
			}

			if message.Is("role", previous) {
				buffer.WriteString(str)
				continue
			}

			tempMessages = append(tempMessages, map[string]string{
				"role":    previous,
				"content": buffer.String(),
			})

			buffer.Reset()
			previous = message.GetString("role")
			buffer.WriteString(str)
		}
	}

	if buffer.Len() > 0 {
		tempMessages = append(tempMessages, map[string]string{
			"role":    previous,
			"content": buffer.String(),
		})
	}

	buffer = new(bytes.Buffer)
	messageL := len(tempMessages)
	previous = "start"
	for idx, message := range tempMessages {
		next := "end"
		if idx+1 < messageL-1 {
			next = tempMessages[idx+1]["role"]
		}

		if buffer.Len() != 0 {
			buffer.WriteByte('\n')
		}

		nextMessages, err := iterator(struct {
			Previous string
			Next     string
			Message  map[string]string
			Buffer   *bytes.Buffer
			Initial  func() pkg.Keyv[interface{}]
		}{Previous: previous, Next: next, Message: message, Buffer: buffer, Initial: func() pkg.Keyv[interface{}] {
			if id, ok := message["id"]; ok {
				return sources[id]
			}
			return nil
		}})

		if err != nil {
			return nil, err
		}

		newMessages = append(newMessages, nextMessages...)
		previous = message["role"]
	}
	return
}

// 单文本内容合并
func MergeStrMessage[T any](messages []T, iter func(message T) string) string {
	buffer := new(bytes.Buffer)
	for _, message := range messages {
		str := iter(message)
		buffer.WriteString(str)
	}
	return buffer.String()
}

func MergeMultiMessage(ctx context.Context, proxies string, message pkg.Keyv[interface{}]) (string, error) {
	contents := make([]string, 0)
	values := message.GetSlice("content")
	if len(values) == 0 {
		return "", nil
	}

	for _, value := range values {
		var keyv pkg.Keyv[interface{}]
		keyv, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		if keyv.Is("type", "text") {
			contents = append(contents, keyv.GetString("text"))
			continue
		}

		if keyv.Is("type", "image_url") {
			o := keyv.GetKeyv("image_url")
			file := o.GetString("url")
			// base64
			if strings.HasPrefix(file, "data:image/") {
				pos := strings.Index(file, ";")
				if pos == -1 {
					return "", errors.New("invalid base64 url")
				}

				mime := file[5:pos]
				ext, err := MimeToSuffix(mime)
				if err != nil {
					return "", err
				}

				file = file[pos+1:]
				if !strings.HasPrefix(file, "base64,") {
					return "", errors.New("invalid base64 url")
				}

				buffer := new(bytes.Buffer)
				w := multipart.NewWriter(buffer)
				fw, err := w.CreateFormFile("image", "1"+ext)
				if err != nil {
					return "", err
				}

				file, err = SaveBase64(file, ext[1:])
				if err != nil {
					return "", err
				}

				fileBytes, err := os.ReadFile(file)
				if err != nil {
					return "", err
				}
				_, _ = fw.Write(fileBytes)
				_ = w.Close()

				r, err := emit.ClientBuilder(nil).
					Proxies(proxies).
					Context(ctx).
					POST("https://complete-mmx-easy-images.hf.space/upload").
					Header("Content-Type", w.FormDataContentType()).
					Header("Authorization", "Bearer 123").
					Buffer(buffer).
					DoS(http.StatusOK)
				if err != nil {
					text := emit.TextResponse(r)
					logger.Error(text)
					return "", err
				}

				obj, err := emit.ToMap(r)
				if err != nil {
					return "", err
				}

				file = obj["URL"].(string)
			}

			contents = append(contents, fmt.Sprintf("*image*: %s\n----", file))
		}
	}

	if len(contents) == 0 {
		return "", nil
	}

	return strings.Join(contents, "\n\n"), nil
}

// 填充垃圾消息
func PadJunkMessage(length int, message string) (str string) {
	if length <= 0 {
		return message
	}

	tokens := CalcTokens(message)
	if tokens >= length {
		return message
	}

	placeholder := ""
	for {
		content := RandString(100)
		tokens += CalcTokens(content)
		placeholder += content

		if tokens >= length {
			break
		}
	}

	return placeholder + "\n\n\n" + message
}
