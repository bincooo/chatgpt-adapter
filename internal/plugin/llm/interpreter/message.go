package interpreter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const ginTokens = "__tokens__"

func waitResponse(ctx *gin.Context, matchers []common.Matcher, r *http.Response, sse bool) (content string) {
	logger.Info("waitResponse ...")
	created := time.Now().Unix()
	completion := common.GetGinCompletion(ctx)
	toolId := common.GetGinToolValue(ctx).GetString("id")
	toolId = plugin.NameWithTools(toolId, completion.Tools)

	scanner := bufio.NewScanner(r.Body)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if eof {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for {
		if !scanner.Scan() {
			break
		}

		data := scanner.Text()
		logger.Tracef("--------- ORIGINAL MESSAGE ---------")
		logger.Tracef("%s", data)

		if len(data) < 6 || data[:6] != "data: " {
			continue
		}

		data = data[6:]
		if data == "[DONE]" {
			break
		}

		var message pkg.Keyv[interface{}]
		err := json.Unmarshal([]byte(data), &message)
		if err != nil {
			logger.Error(err.Error())
			continue
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
