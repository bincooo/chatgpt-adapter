package interpreter

import (
	"bytes"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
)

func fetch(ctx *gin.Context, proxies string, completion pkg.ChatCompletion) (response *http.Response, tokens int, err error) {
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

	system := ""
	if completion.Messages[0].Is("role", "system") {
		system = completion.Messages[0].GetString("content")
		completion.Messages = completion.Messages[1:]
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
		// 复合消息
		if _, ok := opts.Message["multi"]; ok && role == "user" {
			message := opts.Initial()
			content, e := common.MergeMultiMessage(ctx.Request.Context(), proxies, message)
			if e != nil {
				return nil, e
			}
			opts.Buffer.WriteString(content)
			if condition(role) != condition(opts.Next) {
				result = []pkg.Keyv[interface{}]{
					{
						"role":    condition(role),
						"content": opts.Buffer.String(),
						"type":    "message",
					},
				}
				opts.Buffer.Reset()
			}
			return
		}

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

	if messageL := len(messages); !messages[messageL-1].Is("role", "user") {
		return nil, 0, errors.Errorf("messages[%d] is not `user` role", messageL-1)
	}

	message := messages[len(messages)-1].GetString("content")
	messages = messages[:len(messages)-1]

	obj := map[string]interface{}{
		"messages": messages,
	}

	if system != "" {
		obj["system"] = system
	}

	response, err = emit.ClientBuilder().
		Context(ctx.Request.Context()).
		Proxies(proxies).
		POST(baseUrl+"/settings").
		Body(obj).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}
	logger.Info(emit.TextResponse(response))

	response, err = emit.ClientBuilder().
		Context(ctx.Request.Context()).
		Proxies(proxies).
		POST(baseUrl+"/chat").
		Body(map[string]string{
			"message": message,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	return
}
