package cursor

import (
	"bufio"
	"bytes"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"io"
	"net/http"
	"strconv"
	"time"

	. "chatgpt-adapter/relay/llm/cursor/proto"
)

const ginTokens = "__tokens__"

type chunkError struct {
	E struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Type  string `json:"type"`
			Debug struct {
				Error   string `json:"error"`
				Details struct {
					Title       string `json:"title"`
					Detail      string `json:"detail"`
					IsRetryable bool   `json:"isRetryable"`
				} `json:"details"`
				IsExpected bool `json:"isExpected"`
			} `json:"debug"`
			Value string `json:"value"`
		} `json:"details"`
	} `json:"error"`
}

func (ce chunkError) Error() string {
	message := ce.E.Message
	if len(ce.E.Details) > 0 {
		message = ce.E.Details[0].Debug.Details.Detail
	}
	return fmt.Sprintf("[%s] %s", ce.E.Code, message)
}

func waitMessage(r *http.Response, cancel func(str string) bool) (content string, err error) {
	defer r.Body.Close()
	scanner := newScanner(r.Body)
	for {
		if !scanner.Scan() {
			break
		}

		event := scanner.Text()
		if event == "" {
			continue
		}

		if !scanner.Scan() {
			break
		}

		chunk := scanner.Bytes()
		if len(chunk) == 0 {
			continue
		}

		if event[7:] == "error" {
			var chunkErr chunkError
			err = json.Unmarshal(chunk, &chunkErr)
			if err == nil {
				err = &chunkErr
			}
			return
		}

		if event[7:] == "system" || bytes.Equal(chunk, []byte("{}")) {
			continue
		}

		raw := string(chunk)
		logger.Debug("----- raw -----")
		logger.Debug(raw)
		if len(raw) > 0 {
			content += raw
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, r *http.Response, sse bool) (content string) {
	defer r.Body.Close()
	created := time.Now().Unix()
	logger.Info("waitResponse ...")
	matchers := common.GetGinMatchers(ctx)
	tokens := ctx.GetInt(ginTokens)

	scanner := newScanner(r.Body)
	for {
		if !scanner.Scan() {
			raw := response.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			break
		}
		event := scanner.Text()
		if event == "" {
			continue
		}

		if !scanner.Scan() {
			raw := response.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			break
		}

		chunk := scanner.Bytes()
		if len(chunk) == 0 {
			continue
		}

		if event[7:] == "error" {
			var chunkErr chunkError
			err := json.Unmarshal(chunk, &chunkErr)
			if err == nil {
				err = &chunkErr
			}

			if response.NotSSEHeader(ctx) {
				logger.Error(err)
				response.Error(ctx, -1, err)
			}
			return
		}

		if event[7:] == "system" || bytes.Equal(chunk, []byte("{}")) {
			continue
		}

		raw := string(chunk)
		logger.Debug("----- raw -----")
		logger.Debug(raw)

		raw = response.ExecMatchers(matchers, raw, false)
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

	ctx.Set(vars.GinCompletionUsage, response.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

func echoMessages(ctx *gin.Context, completion model.Completion) {
	content := ""
	var (
		toolMessages = toolcall.ExtractToolMessages(&completion)
	)

	if len(toolMessages) == 1 {
		content = completion.Messages[0].GetString("content")
	} else {
		chunkBytes, _ := json.MarshalIndent(completion.Messages, "", "  ")
		content = string(chunkBytes)
	}

	if len(toolMessages) > 0 {
		content += "\n----------toolCallMessages----------\n"
		chunkBytes, _ := json.MarshalIndent(toolMessages, "", "  ")
		content += string(chunkBytes)
	}

	response.Echo(ctx, completion.Model, content, completion.Stream)
}

func newScanner(body io.ReadCloser) (scanner *bufio.Scanner) {
	// 每个字节占8位
	// 00000011 第一个字节是占位符，应该是用来代表消息类型的 假定 0: 消息体/proto, 1: 系统提示词/gzip, 3: 错误标记/gzip
	// 00000000 00000000 00000010 11011000 4个字节描述包体大小
	scanner = bufio.NewScanner(body)
	var (
		magic    byte
		chunkLen int64 = -1
		setup          = 5
	)

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return
		}

		if atEOF {
			return len(data), data, err
		}

		if chunkLen == -1 && len(data) < setup {
			return
		}

		if chunkLen == -1 {
			magic = data[0]
			chunk := hex.EncodeToString(data[1:setup])
			chunkLen, err = strconv.ParseInt(chunk, 16, 64)
			if err != nil {
				return
			}

			// 这部分应该是分割标记？或者补位
			if magic == 0 && chunkLen == 0 {
				chunkLen = -1
				return 5, []byte(""), err
			}

			if magic == 3 { // 假定它是错误标记
				return setup, []byte("event: error"), err
			}

			if magic == 1 { // 系统提示词标记？
				return setup, []byte("event: system"), err
			}
			// magic == 0
			return setup, []byte("event: message"), err
		}

		if int64(len(data)) < chunkLen {
			return
		}

		chunk := data[:chunkLen]
		chunkLen = -1

		i := len(chunk)
		// 解码
		if emit.IsEncoding(chunk, "gzip") {
			reader, gzErr := emit.DecodeGZip(io.NopCloser(bytes.NewReader(chunk)))
			if gzErr != nil {
				err = gzErr
				return
			}
			chunk, err = io.ReadAll(reader)
		}
		if magic == 0 {
			var message ResMessage
			err = proto.Unmarshal(chunk, &message)
			if err != nil {
				return
			}
			chunk = []byte(message.Msg)
		}
		return i, chunk, err
	})

	return
}
