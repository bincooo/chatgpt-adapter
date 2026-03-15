package arena

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xllm-go/g/interceptor"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

func createChannel(ctx *model.Ctx, reader io.Reader) chan *model.ChunkBodies {
	channel := make(chan *model.ChunkBodies, 1)
	completion := ctx.GetCompletion()
	logger.Sugar().Infof("create new channel")

	go func() {
		scanner := bufio.NewScanner(reader)
		defer func() {
			chunk := interceptor.ExecuteInterceptors(ctx, "", true)
			if chunk != "" {
				channel <- &model.ChunkBodies{Chunk: chunk, Stream: true}
			}
			close(channel)
		}()

		callers := make([]func(), 0)
		callers = append(callers, func() {
			if think, ok := model.GetValue[string, string](ctx.Record, interceptor.ThinkReason); ok {
				ctx.Record.Del(interceptor.ThinkReason)
				channel <- &model.ChunkBodies{Think: think, Stream: true}
			}
		})
		callers = append(callers, func() {
			if chunk, ok := model.GetValue[string, string](ctx.Record, interceptor.ToolCall); ok {
				channel <- model.CreateFunction(chunk, completion.Stream)
				ctx.Cancel()
			}
		})

		for {
			select {
			case <-ctx.Context().Done():
				return
			default:

				if ok := scan(ctx, scanner, channel, callers...); ok {
					return
				}
			}
		}
	}()

	return channel
}

func scan(ctx *model.Ctx, scanner *bufio.Scanner, channel chan *model.ChunkBodies, calls ...func()) (ok bool) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			logger.Sugar().Error(err)
			channel <- &model.ChunkBodies{Chunk: fmt.Sprintf("error: %v", err), Stream: true}
		}
		ok = true
		return
	}

	data := scanner.Text()
	if len(data) < 3 {
		return
	}

	if strings.HasPrefix(data, "{\"error\":") {
		channel <- &model.ChunkBodies{Chunk: fmt.Sprintf("error: %v", data), Stream: true}
		ok = true
		return
	}

	state := data[:2]
	data = data[3:]
	if state == "ad" && strings.Contains(data, "\"finishReason\":\"stop\"") {
		return
	}

	if state == "a2" {
		return
	}

	if len(data) == 0 {
		return
	}

	chunk, err := strconv.Unquote(data)
	if err != nil {
		logger.Sugar().Errorf("转义失败： %v -- %s", err, data)
		return
	}

	logger.Sugar().Debug("----- raw -----")
	logger.Sugar().Debug(chunk)
	if state == "ag" {
		each(chunk, func(message string) {
			channel <- &model.ChunkBodies{Think: message, Stream: true}
		})
		return
	}

	chunk = interceptor.ExecuteInterceptors(ctx, chunk, false)
	for _, yield := range calls {
		yield()
	}
	if chunk == "" {
		return
	}

	each(chunk, func(message string) {
		channel <- &model.ChunkBodies{Chunk: message, Stream: true}
	})
	return
}

func each(content string, w func(chunk string)) {
	pos := 0
	runeStr := []rune(content)
	step := 30

	for {
		contentL := len(runeStr[pos:])
		if contentL > step {
			w(string(runeStr[pos : pos+step]))
			pos += step
			continue
		}

		w(string(runeStr[pos:]))
		time.Sleep(100 * time.Millisecond)
		break
	}
}
