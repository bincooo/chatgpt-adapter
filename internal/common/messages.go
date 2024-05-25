package common

import (
	"bytes"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/google/uuid"
	"strings"
)

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

func StringCombiner[T any](messages []T, iter func(message T) string) string {
	buffer := new(bytes.Buffer)
	for _, message := range messages {
		str := iter(message)
		buffer.WriteString(str)
	}
	return buffer.String()
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
