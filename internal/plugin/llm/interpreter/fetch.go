package interpreter

import (
	"bytes"
	"fmt"
	"github.com/RomiChan/websocket"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"net/http"
)

var (
	start = []byte{123, 34, 114, 111, 108, 101, 34, 58, 32, 34, 117, 115, 101, 114, 34, 44, 32, 34, 116, 121, 112, 101, 34, 58, 32, 34, 109, 101, 115, 115, 97, 103, 101, 34, 44, 32, 34, 115, 116, 97, 114, 116, 34, 58, 32, 116, 114, 117, 101, 125}
	end   = []byte{123, 34, 114, 111, 108, 101, 34, 58, 32, 34, 117, 115, 101, 114, 34, 44, 32, 34, 116, 121, 112, 101, 34, 58, 32, 34, 109, 101, 115, 115, 97, 103, 101, 34, 44, 32, 34, 101, 110, 100, 34, 58, 32, 116, 114, 117, 101, 125}
)

func fetch(ctx *gin.Context, proxies string, completion pkg.ChatCompletion) (conn *websocket.Conn, tokens int, err error) {
	var (
		baseSocket = pkg.Config.GetString("interpreter.baseSocket")
		useProxy   = pkg.Config.GetBool("interpreter.useProxy")
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

	response, err := emit.ClientBuilder().
		Context(ctx.Request.Context()).
		Proxies(proxies).
		POST(replace(baseSocket)+"/settings").
		Body(map[string]interface{}{
			"messages": messages,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}
	logger.Info(emit.TextResponse(response))

	conn, err = emit.SocketBuilder().
		Context(ctx.Request.Context()).
		Proxies(proxies).
		URL(baseSocket).
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		return
	}

	err = sendMessage(conn, message)
	return
}

func sendMessage(conn *websocket.Conn, message string) (err error) {
	err = conn.WriteMessage(websocket.TextMessage, start)
	if err != nil {
		return
	}

	err = conn.WriteJSON(map[string]interface{}{
		"role":    "user",
		"type":    "message",
		"content": message,
	})
	if err != nil {
		return
	}

	return conn.WriteMessage(websocket.TextMessage, end)
}

func replace(bu string) string {
	if len(bu) > 5 && bu[:5] == "ws://" {
		return "http://" + bu[5:]
	}
	if len(bu) > 6 && bu[:6] == "wss://" {
		return "https://" + bu[6:]
	}
	return bu
}
