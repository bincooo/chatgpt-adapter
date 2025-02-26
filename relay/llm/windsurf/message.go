package windsurf

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/iocgo/sdk/env"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
)

const (
	ginTokens = "__tokens__"
	thinkTag  = "think: "
)

type ChunkErrorWrapper struct {
	Cause struct {
		Wrapper *ChunkErrorWrapper `json:"wrapper"`
		Leaf    struct {
			Message string `json:"message"`
			Details struct {
				OriginalTypeName string `json:"originalTypeName"`
				ErrorTypeMark    struct {
					FamilyName string `json:"familyName"`
				} `json:"errorTypeMark"`
			} `json:"details"`
		} `json:"leaf"`
	} `json:"cause"`
	Message string `json:"message"`
	Details struct {
		OriginalTypeName string `json:"originalTypeName"`
		ErrorTypeMark    struct {
			FamilyName string `json:"familyName"`
		} `json:"errorTypeMark"`
		ReportablePayload []string `json:"reportablePayload"`
		FullDetails       struct {
			Type string `json:"@type"`
			Msg  string `json:"msg"`
		} `json:"fullDetails"`
	} `json:"details"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
		Debug struct {
			Wrapper *ChunkErrorWrapper `json:"wrapper"`
		} `json:"debug"`
	} `json:"details"`
}

type chunkError struct {
	E Error `json:"error"`
}

func (ce chunkError) Error() string {
	return ce.E.Error()
}

func (e Error) Error() string {
	message := e.Message
	if len(e.Details) > 0 {
		wrapper := e.Details[0].Debug.Wrapper
		for {
			if wrapper == nil {
				break
			}
			if wrapper.Cause.Wrapper == nil {
				break
			}
			wrapper = wrapper.Cause.Wrapper
		}
		if wrapper != nil {
			message = wrapper.Cause.Leaf.Message
		}
	}
	return fmt.Sprintf("[%s] %s", e.Code, message)
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
			if bytes.Equal(chunk, []byte("{}")) {
				break
			}
			var chunkErr chunkError
			err = json.Unmarshal(chunk, &chunkErr)
			if err == nil {
				err = &chunkErr
			}
			return
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
	completion := common.GetGinCompletion(ctx)
	matchers := common.GetGinMatchers(ctx)
	tokens := ctx.GetInt(ginTokens)
	thinkReason := env.Env.GetBool("server.think_reason")
	thinkReason = thinkReason && completion.Model[9:] == "deepseek-reasoner"
	reasoningContent := ""
	think := 0

	onceExec := sync.OnceFunc(func() {
		if !sse {
			ctx.Writer.WriteHeader(http.StatusOK)
		}
	})

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
			if bytes.Equal(chunk, []byte("{}")) {
				break
			}
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

		raw := string(chunk)
		reasonContent := ""
		if strings.HasPrefix(raw, thinkTag) {
			reasonContent = raw[len(thinkTag):]
			raw = ""
			think = 2
			logger.Debug("----- think raw -----")
			logger.Debug(reasonContent)
			reasoningContent += reasonContent
			goto label

		} else if thinkReason && think == 0 {
			if strings.HasPrefix(raw, "<think>") {
				reasonContent = raw[7:]
				raw = ""
				think = 1
				goto label
			}
		}

		if thinkReason && think == 1 {
			reasonContent = raw
			if strings.HasPrefix(raw, "</think>") {
				reasonContent = ""
				think = 2
			}

			raw = ""
			logger.Debug("----- think raw -----")
			logger.Debug(reasonContent)
			reasoningContent += reasonContent
			goto label
		}

		logger.Debug("----- raw -----")
		logger.Debug(raw)
		onceExec()

		raw = response.ExecMatchers(matchers, raw, false)
		if len(raw) == 0 {
			continue
		}

		if raw == response.EOF {
			break
		}

	label:
		if sse {
			response.ReasonSSEResponse(ctx, Model, raw, reasonContent, created)
		}
		content += raw
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	ctx.Set(vars.GinCompletionUsage, response.CalcUsageTokens(content, tokens))
	if !sse {
		response.ReasonResponse(ctx, Model, content, reasoningContent)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

func newScanner(body io.ReadCloser) (scanner *bufio.Scanner) {
	// 每个字节占8位
	// 00000011 第一个字节是占位符，应该是用来代表消息类型的 假定 1: 消息体/proto+gzip, 3: 错误标记/gzip
	// 00000000 00000000 00000010 11011000 4个字节描述包体大小
	scanner = bufio.NewScanner(body)
	var (
		magic    byte
		chunkLen = -1
		setup    = 5
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
			chunkLen = bytesToInt32(data[1:setup])

			if magic == 3 { // 假定它是错误标记 ?? 有没搞错，error和done同用一个标记？
				return setup, []byte("event: error"), err
			}

			// magic == 1
			return setup, []byte("event: message"), err
		}

		if len(data) < chunkLen {
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

		if magic != 3 {
			var message ResMessage
			err = proto.Unmarshal(chunk, &message)
			if err != nil {
				return
			}

			if message.Think != "" {
				chunk = append([]byte(thinkTag), []byte(message.Think)...)
			} else {
				chunk = []byte(message.Message)
			}
		}
		return i, chunk, err
	})

	return
}
