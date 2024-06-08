package interpreter

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"net/http"
)

func fetch(ctx context.Context, proxies string, completion pkg.ChatCompletion) (r *http.Response, tokens int, err error) {
	var (
		baseUrl  = pkg.Config.GetString("interpreter.baseUrl")
		useProxy = pkg.Config.GetBool("interpreter.useProxy")
	)

	if !useProxy {
		proxies = ""
	}

	condition := func(role string) string {
		switch role {
		case "assistant", "end":
			return role
		default:
			return "user"
		}
	}

	messages, _ := common.TextMessageCombiner(completion.Messages, func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (result []pkg.Keyv[interface{}], err error) {
		role := opts.Message["role"]
		tokens += common.CalcTokens(opts.Message["content"])

		if condition(role) == condition(opts.Next) {
			// cache buffer
			opts.Buffer.WriteString(opts.Message["content"])
			return
		}

		defer opts.Buffer.Reset()
		opts.Buffer.WriteString(fmt.Sprintf(opts.Message["content"]))
		result = []pkg.Keyv[interface{}]{
			{
				"role":    condition(role),
				"content": opts.Buffer.String(),
				"type":    "message",
			},
		}
		return
	})

	message := messages[len(messages)-1].GetString("content")
	messages = messages[:len(messages)-1]

	r, err = emit.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		POST(baseUrl+"/chat").
		JHeader().
		Body(map[string]interface{}{
			"previousMessages": messages,
			"message":          message,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	return
}
