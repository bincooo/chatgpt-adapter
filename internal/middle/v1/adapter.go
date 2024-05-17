package v1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
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
	middle.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	return Model == model
}

func (API) Models() []middle.Model {
	return []middle.Model{
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

	response, err := fetchGpt35(ctx, completion)
	if err != nil {
		code := -1
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			code = http.StatusTooManyRequests
			go common.ChangeClashIP()
		}
		middle.ErrResponse(ctx, code, err)
		return
	}
	waitResponse(ctx, response, matchers, completion.Stream)
}

func waitMessage(response *http.Response, cancel func(str string) bool) (content string, err error) {
	scanner := bufio.NewScanner(response.Body)
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

func waitResponse(ctx *gin.Context, response *http.Response, matchers []pkg.Matcher, sse bool) {
	tokens := ctx.GetInt("tokens")
	scanner := bufio.NewScanner(response.Body)
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
			middle.ErrResponse(ctx, -1, err)
			return
		}

		if r.Error != nil {
			logrus.Errorf("%v", r.Error)
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
		fmt.Printf("----- raw -----\n %s\n", raw)
		raw = pkg.ExecMatchers(matchers, raw)

		if sse {
			middle.SSEResponse(ctx, Model, raw, nil, created)
		}
	}

	if !sse {
		middle.Response(ctx, Model, content)
	} else {
		middle.SSEResponse(ctx, Model, "[DONE]", common.CalcUsageTokens(content, tokens), created)
	}
}
