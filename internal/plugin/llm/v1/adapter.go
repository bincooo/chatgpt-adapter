package v1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/gin-gonic/gin"
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
	if completion.Model == "freeGpt35" {
		completion.Model = "text-davinci-002-render-sha"
	}

	r, err := fetchGpt35(ctx, completion)
	if err != nil {
		code := -1
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			code = http.StatusTooManyRequests
			go common.ChangeClashIP()
		}
		response.Error(ctx, code, err)
		return
	}
	waitResponse(ctx, r, matchers, completion.Stream)
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

func waitResponse(ctx *gin.Context, r *http.Response, matchers []common.Matcher, sse bool) {
	tokens := ctx.GetInt("tokens")
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
	content := ""
	created := time.Now().Unix()

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
		if err := json.Unmarshal([]byte(text[5:]), &r); err != nil {
			response.Error(ctx, -1, err)
			return
		}

		if r.Error != nil {
			logger.Errorf("%v", r.Error)
			return
		}

		if r.Message.Author.Role != "assistant" {
			continue
		}

		if len(r.Message.Content.Parts) == 0 || len(r.Message.Content.Parts[0]) == 0 {
			continue
		}

		raw := r.Message.Content.Parts[0]
		if len(raw) <= pos {
			continue
		}
		content = raw
		raw = raw[pos:]

		logger.Debug("----- raw -----")
		logger.Debug(raw)

		raw = common.ExecMatchers(matchers, raw)

		if sse {
			response.SSEResponse(ctx, Model, raw, created)
		}
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
}
