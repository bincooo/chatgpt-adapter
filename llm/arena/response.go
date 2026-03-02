package arena

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"

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
			return model.CreateFunction(bodies.Function.Name, bodies.Function.Args)
		}
	}
	return model.CreateChunk(chunk, think)
}

func createChannel(ctx *model.Ctx, reader io.Reader) chan *model.ChunkBodies {
	channel := make(chan *model.ChunkBodies)
	completion := model.JustValue[string, *model.Completion](ctx.Record, "completion")

	go func() {
		scanner := bufio.NewScanner(reader)
		defer func() {
			chunk := model.ExecMatchers(ctx, "", true)
			if chunk != "" {
				channel <- model.CreateChunk(chunk, "")
			}
			close(channel)
		}()

		for {
			select {
			case <-ctx.Ctx().Context().Done():
				return
			default:
				if ok := scan(ctx, scanner, channel,
					sync.OnceFunc(func() {
						if think, ok := model.GetValue[string, string](ctx.Record, model.ThinkReason); ok {
							channel <- model.CreateChunk("", think)
						}
					}),
					sync.OnceFunc(func() {
						if chunk, ok := model.GetValue[string, string](ctx.Record, model.ToolCall); ok {
							var fc model.FuncCall
							_ = json.Unmarshal([]byte(chunk), &fc)
							if completion.Stream {
								channel <- model.CreateFunction(fc.Name, make(json.RawMessage, 0))
							}
							channel <- model.CreateFunction(fc.Name, fc.Args)
							ctx.Cancel()
						}
					}),
				); ok {
					return
				}
			}
		}
	}()

	return channel
}

func scan(ctx *model.Ctx, scanner *bufio.Scanner, channel chan *model.ChunkBodies, onceSlice ...func()) (ok bool) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			logger.Sugar().Error(err)
			channel <- model.CreateChunk("error: "+err.Error(), "")
		}
		ok = true
		return
	}

	data := scanner.Text()
	if len(data) < 3 {
		return
	}

	state := data[:2]
	data = data[3:]
	if state == "ad" && strings.Contains(data, "\"finishReason\":\"stop\"") {
		ok = true
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
	chunk = model.ExecMatchers(ctx, chunk, false)
	for _, yield := range onceSlice {
		yield()
	}
	if chunk == "" {
		return
	}

	model.SplitEach(chunk, func(message string) {
		channel <- model.CreateChunk(message, "")
	})
	return
}
