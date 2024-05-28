package v1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

var (
	Adapter = API{}
	Model   = "freeGpt35"
)

type API struct {
	plugin.BaseAdapter
}

const ginTokens = "__tokens__"

func (API) Match(_ *gin.Context, model string) bool {
	return Model == model
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "freeGpt35",
			Object:  "model",
			Created: 1686935002,
			By:      "chatgpt-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, completion) {
			return
		}
	}

	messages, tokens := mergeMessages(completion.Messages)
	ctx.Set(ginTokens, tokens)
	r, err := fetchGpt35(ctx, messages)
	if err != nil {
		code := -1
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			code = http.StatusTooManyRequests
			go common.ChangeClashIP()
		}
		logger.Error(err)
		response.Error(ctx, code, err)
		return
	}

	content := waitResponse(ctx, r, matchers, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func waitMessage(r *http.Response, cancel func(str string) bool) (content string, err error) {
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

		text := scanner.Text()
		if len(text) < 5 || text[:5] != "data:" {
			continue
		}

		if text == "data: [DONE]" {
			break
		}

		var r chatSSEResponse
		if err = json.Unmarshal([]byte(text[5:]), &r); err != nil {
			return
		}

		if r.Error != nil {
			return "", fmt.Errorf("%v", r.Error)
		}

		if r.Message.Author.Role != "assistant" {
			continue
		}

		if len(r.Message.Content.Parts) == 0 || len(r.Message.Content.Parts[0]) == 0 {
			continue
		}

		content = r.Message.Content.Parts[0]
		if cancel != nil && cancel(content) {
			return
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, r *http.Response, matchers []common.Matcher, sse bool) (content string) {
	tokens := ctx.GetInt(ginTokens)
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

	pos := 0
	created := time.Now().Unix()

	for {
		if !scanner.Scan() {
			break
		}

		text := scanner.Text()
		logrus.Tracef("--------- ORIGINAL MESSAGE ---------")
		logrus.Trace(text)

		if len(text) < 5 || text[:5] != "data:" {
			continue
		}

		if text == "data: [DONE]" {
			break
		}

		var res chatSSEResponse
		if err := json.Unmarshal([]byte(text[5:]), &res); err != nil {
			logger.Error(err)
			response.Error(ctx, -1, err)
			return
		}

		if res.Error != nil {
			logger.Errorf("%v", res.Error)
			return
		}

		if res.Message.Author.Role != "assistant" {
			continue
		}

		if len(res.Message.Content.Parts) == 0 || len(res.Message.Content.Parts[0]) == 0 {
			continue
		}

		raw := res.Message.Content.Parts[0]
		if len(raw) <= pos {
			continue
		}
		content = raw
		raw = raw[pos:]

		logger.Debug("----- raw -----")
		logger.Debug(raw)

		raw = common.ExecMatchers(matchers, raw)
		if len(raw) == 0 {
			continue
		}

		if sse {
			response.SSEResponse(ctx, Model, raw, created)
		}
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}
