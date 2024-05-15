package v1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

const MODEL = "freeGpt35"

// ChatGPT
func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []common.Matcher) {
	if req.Model == "freeGpt35" {
		req.Model = "text-davinci-002-render-sha"
	}

	response, err := fetchGpt35(ctx, req)
	if err != nil {
		code := -1
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			code = http.StatusTooManyRequests
			go common.ChangeClashIP()
		}
		middle.ResponseWithE(ctx, code, err)
		return
	}
	resolve(ctx, response, matchers, req.Stream)
}

func resolve(ctx *gin.Context, response *http.Response, matchers []common.Matcher, sse bool) {
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
			middle.ResponseWithE(ctx, -1, err)
			return
		}

		if r.Error != nil {
			middle.ResponseWithV(ctx, -1, fmt.Sprintf("%v", r.Error))
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
		raw = common.ExecMatchers(matchers, raw)

		if sse {
			middle.ResponseWithSSE(ctx, MODEL, raw, nil, created)
		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", common.CalcUsageTokens(content, tokens), created)
	}
}
