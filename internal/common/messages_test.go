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
	nMessages, _ := TextMessageCombiner[map[string]string](messages, func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) ([]map[string]string, error) {
		role := opts.Message["role"]
		if condition(role) != condition(opts.Next) {
			if opts.Buffer.Len() != 0 {
				opts.Buffer.WriteByte('\n')
			}
			opts.Buffer.WriteString(opts.Message["content"])
			defer opts.Buffer.Reset()
			return []map[string]string{
				{
					"role":    condition(role),
					"content": opts.Buffer.String(),
				},
			}, nil
		}

		if opts.Buffer.Len() != 0 {
			opts.Buffer.WriteByte('\n')
		}
		opts.Buffer.WriteString(opts.Message["content"])
		return nil, nil
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
