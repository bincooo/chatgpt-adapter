package common

import (
	"bytes"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"math/rand"
	"strings"
	"time"
)

func MessageCombiner[T any](
	messages []pkg.Keyv[interface{}],
	iter func(previous, next string, message map[string]string, buffer *bytes.Buffer) []T,
) (newMessages []T) {

	previous := "start"
	buffer := new(bytes.Buffer)
	msgs := make([]map[string]string, 0)
	for _, message := range messages {
		str := strings.TrimSpace(message.GetString("content"))
		if str == "" {
			continue
		}

		if buffer.Len() != 0 {
			buffer.WriteString("\n\n")
		}

		if previous == "start" {
			previous = message.GetString("role")
			buffer.WriteString(str)
			continue
		}

		if previous == message["role"] {
			buffer.WriteString(str)
			continue
		}

		msgs = append(msgs, map[string]string{
			"role":    previous,
			"content": buffer.String(),
		})

		buffer.Reset()
		previous = message.GetString("role")
		buffer.WriteString(str)
	}

	if buffer.Len() > 0 {
		msgs = append(msgs, map[string]string{
			"role":    previous,
			"content": buffer.String(),
		})
	}

	buffer = new(bytes.Buffer)
	messageL := len(msgs)
	previous = "start"
	for idx, message := range msgs {
		next := "end"
		if idx+1 < messageL-1 {
			next = msgs[idx+1]["role"]
		}

		if buffer.Len() != 0 {
			buffer.WriteByte('\n')
		}

		nextMessages := iter(previous, next, message, buffer)
		if len(next) > 0 {
			newMessages = append(newMessages, nextMessages...)
		}

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

func NeedToToolCall(completion pkg.ChatCompletion) bool {
	messageL := len(completion.Messages)
	if messageL == 0 {
		return false
	}

	if len(completion.Tools) == 0 {
		return false
	}

	return completion.Messages[messageL-1]["role"] != "function"
}

func PadText(length int, message string) string {
	if length <= 0 {
		return message
	}

	s := "abcdefghijklmnopqrstuvwsyz0123456789!@#$%^&*()_+,.?/\\"
	bin := make([]byte, length)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for idx := 0; idx < length; idx++ {
		pos := r.Intn(len(s))
		u := s[pos]
		bin[idx] = u
	}

	layout := "%s\n------\n\n%s"
	return fmt.Sprintf(layout, string(bin), message)
}
