package arena

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

func waitChannel(ctx *model.Ctx, response io.Reader) *model.ChunkBodies {
	channel := createChannel(ctx, response)
	var chunk, think string
	for {
		bodies, ok := <-channel
		if !ok {
			break
		}
		chunk += bodies.Chunk
		if bodies.Think != "" {
			think = bodies.Think
		}
		if bodies.Function != nil {
			bodies.Stream = false
			return bodies
		}
	}

	return &model.ChunkBodies{Chunk: chunk, Think: think}
}

func createChannel(ctx *model.Ctx, reader io.Reader) chan *model.ChunkBodies {
	channel := make(chan *model.ChunkBodies, 1)
	completion := model.JustValue[string, *model.Completion](ctx.Record, "completion")

	go func() {
		scanner := bufio.NewScanner(reader)
		defer func() {
			chunk := model.ExecMatchers(ctx, "", true)
			if chunk != "" {
				channel <- &model.ChunkBodies{Chunk: chunk, Stream: true}
			}
			close(channel)
		}()

		for {
			select {
			case <-ctx.Context().Done():
				return
			default:
				calls := make([]func(), 0)
				calls = append(calls, sync.OnceFunc(func() {
					if think, ok := model.GetValue[string, string](ctx.Record, model.ThinkReason); ok {
						channel <- &model.ChunkBodies{Think: think, Stream: true}
					}
				}))
				calls = append(calls, sync.OnceFunc(func() {
					if chunk, ok := model.GetValue[string, string](ctx.Record, model.ToolCall); ok {
						channel <- model.CreateFunction(chunk, completion.Stream)
						ctx.Cancel()
					}
				}))
				if ok := scan(ctx, scanner, channel, calls...); ok {
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
		channel <- &model.ChunkBodies{Err: errors.New(data), Stream: true}
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
		logger.Sugar().Errorf("转义失败： %v -- %s", err, chunk)
		return
	}

	logger.Sugar().Debug("----- raw -----")
	logger.Sugar().Debug(chunk)
	if state == "ag" {
		splitEach(chunk, func(message string) {
			channel <- &model.ChunkBodies{Think: message, Stream: true}
		})
		return
	}

	chunk = model.ExecMatchers(ctx, chunk, false)
	for _, yield := range calls {
		yield()
	}
	if chunk == "" {
		return
	}

	splitEach(chunk, func(message string) {
		channel <- &model.ChunkBodies{Chunk: message, Stream: true}
	})
	return
}

func splitEach(content string, w func(chunk string)) {
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
		time.Sleep(80 * time.Millisecond)
		break
	}
}
