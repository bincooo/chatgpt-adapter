package interpreter

import (
	"encoding/json"
	"fmt"
	"github.com/RomiChan/websocket"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
	"time"
)

const ginTokens = "__tokens__"

func waitResponse(ctx *gin.Context, matchers []common.Matcher, conn *websocket.Conn, sse bool) (content string) {
	logger.Info("waitResponse ...")
	created := time.Now().Unix()
	completion := common.GetGinCompletion(ctx)
	toolId := common.GetGinToolValue(ctx).GetString("id")
	toolId = plugin.NameWithTools(toolId, completion.Tools)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			logger.Error(err)
			if response.NotSSEHeader(ctx) {
				response.Response(ctx, Model, err.Error())
				return
			}
			break
		}

		logger.Tracef("--------- ORIGINAL MESSAGE ---------")
		logger.Tracef("%s", data)

		var message pkg.Keyv[interface{}]
		err = json.Unmarshal(data, &message)
		if err != nil {
			logger.Error(err)
			continue
		}

		// DONE
		if message.Is("role", "server") && message.Is("content", "DONE") {
			ctx.Set(vars.GinClose, true)
			_ = conn.WriteMessage(websocket.CloseMessage, nil)
			break
		}

		if !message.Is("role", "assistant") || !message.In("type", "message", "code") {
			continue
		}

		raw := message.GetString("content")

		if message.Is("type", "code") && message.Is("start", true) {
			raw += fmt.Sprintf("\n```%s\n", message.GetString("format"))
		}

		if message.Is("type", "code") && message.Is("end", true) {
			raw += "\n```\n"
		}

		if len(raw) == 0 {
			continue
		}

		logger.Debug("----- raw -----")
		logger.Debug(raw)

		raw = common.ExecMatchers(matchers, raw)
		if len(raw) == 0 {
			continue
		}

		if sse && len(raw) > 0 {
			response.SSEResponse(ctx, Model, raw, created)
		}
		content += raw
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}
