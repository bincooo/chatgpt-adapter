package interpreter

import (
	"bytes"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
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

	response, err = emit.ClientBuilder().
		Context(ctx.Request.Context()).
		Proxies(proxies).
		POST(baseUrl+"/settings").
		Body(map[string]interface{}{
			"messages": messages,
		}).
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
