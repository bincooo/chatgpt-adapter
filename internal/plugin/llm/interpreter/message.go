package interpreter

import (
	"bufio"
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const ginTokens = "__tokens__"

func waitResponse(ctx *gin.Context, matchers []common.Matcher, r *http.Response, sse bool) (content string) {
	defer r.Body.Close()

	logger.Info("waitResponse ...")
	created := time.Now().Unix()
	completion := common.GetGinCompletion(ctx)
	toolId := common.GetGinToolValue(ctx).GetString("id")
	toolId = plugin.NameWithTools(toolId, completion.Tools)
	echoCode := pkg.Config.GetBool("interpreter.echoCode")

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
			if err := scanner.Err(); err != nil {
				logger.Error(err)
			}
			break
		}

		data := scanner.Text()
		logger.Tracef("--------- ORIGINAL MESSAGE ---------")
		logger.Tracef("%s", data)

		if len(data) < 6 || data[:6] != "data: " {
			continue
		}

		data = data[6:]
		if data == "[DONE]" || data == "[DONE]\r" {
			raw := common.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			break
		}

		var message pkg.Keyv[interface{}]
		err := json.Unmarshal([]byte(data), &message)
		if err != nil {
			logger.Error(err)
			continue
		}

		if !message.Is("role", "assistant") || !message.In("type", "message", "code") {
			continue
		}

		// 控制是否输出代码
		if !echoCode && message.Is("type", "code") {
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

		raw = common.ExecMatchers(matchers, raw, false)
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

func waitResponseWS(ctx *gin.Context, matchers []common.Matcher, sse bool) (content string) {
	defer close(wsChan)

	logger.Info("waitResponse ...")
	created := time.Now().Unix()
	completion := common.GetGinCompletion(ctx)
	toolId := common.GetGinToolValue(ctx).GetString("id")
	toolId = plugin.NameWithTools(toolId, completion.Tools)
	echoCode := pkg.Config.GetBool("interpreter.echoCode")

	// return true 结束
	handler := func() bool {
		data, ok := <-wsChan
		if !ok {
			raw := common.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			return true
		}

		logger.Tracef("--------- ORIGINAL MESSAGE ---------")
		logger.Tracef("%s", data)

		//if len(data) < 6 || data[:6] != "data: " {
		//	return false
		//}
		//
		//data = data[6:]
		if data == "[DONE]" || data == "[DONE]\r" {
			return true
		}

		var message pkg.Keyv[interface{}]
		err := json.Unmarshal([]byte(data), &message)
		if err != nil {
			logger.Error(err)
			return false
		}

		if !message.Is("role", "assistant") || !message.In("type", "message", "code") {
			return false
		}

		// 控制是否输出代码
		if !echoCode && message.Is("type", "code") {
			return false
		}

		raw := message.GetString("content")

		if message.Is("type", "code") && message.Is("start", true) {
			raw += fmt.Sprintf("\n```%s\n", message.GetString("format"))
		}

		if message.Is("type", "code") && message.Is("end", true) {
			raw += "\n```\n"
		}

		if len(raw) == 0 {
			return false
		}

		logger.Debug("----- raw -----")
		logger.Debug(raw)

		raw = common.ExecMatchers(matchers, raw, false)
		if len(raw) == 0 {
			return false
		}

		if sse && len(raw) > 0 {
			response.SSEResponse(ctx, Model, raw, created)
		}
		content += raw
		return false
	}

	c := common.GetGinContext(ctx)
	for {
		select {
		case <-c.Done():
			logger.Error("timed out [ws] waitResponse")
			_ = ws.Emit("cancel")
			goto label
		default:
			if handler() {
				goto label
			}
		}
	}

label:
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
