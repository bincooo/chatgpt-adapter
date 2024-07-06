package common

import (
	"bytes"
	"chatgpt-adapter/pkg"
	"fmt"
	"testing"
)

func TestMessageCombiner(t *testing.T) {
	messages := []pkg.Keyv[interface{}]{
		{
			"role":    "user",
			"content": "hello~",
		},
		{
			"role":    "user",
			"content": "hi.",
		},
		{
			"role":    "assistant",
			"content": "bye~",
		},
		{
			"role":    "system",
			"content": "I'm system~",
		},
	}

	condition := func(expr string) string {
		switch expr {
		case "user", "system":
			return "user"
		case "assistant":
			return expr
		default:
			return ""
		}
	}
	nMessages := TextMessageCombiner[map[string]string](messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []map[string]string {
		role := message["role"]
		if condition(role) != condition(next) {
			if buffer.Len() != 0 {
				buffer.WriteByte('\n')
			}
			buffer.WriteString(message["content"])
			defer buffer.Reset()
			return []map[string]string{
				{
					"role":    condition(role),
					"content": buffer.String(),
				},
			}
		}

		if buffer.Len() != 0 {
			buffer.WriteByte('\n')
		}
		buffer.WriteString(message["content"])
		return nil
	})

	for _, msg := range nMessages {
		fmt.Printf("<|%s|>\n%s\n<|end|>", msg["role"], msg["content"])
		fmt.Println()
	}
	t.Log("over")
}

func TestPadJunkMessage(t *testing.T) {
	println(PadJunkMessage(100, "hi"))
}
