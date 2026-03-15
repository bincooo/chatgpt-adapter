package nexos

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"encoding/json"

	"github.com/xllm-go/g/interceptor"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

func createChannel(ctx *model.Ctx, reader io.Reader) chan *model.ChunkBodies {
	channel := make(chan *model.ChunkBodies, 1)
	completion := ctx.GetCompletion()

	go func() {
		scanner := bufio.NewScanner(reader)
		defer func() {
			chunk := interceptor.ExecuteInterceptors(ctx, "", true)
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
					if think, ok := model.GetValue[string, string](ctx.Record, interceptor.ThinkReason); ok {
						channel <- &model.ChunkBodies{Think: think, Stream: true}
					}
				}))
				calls = append(calls, sync.OnceFunc(func() {
					if chunk, ok := model.GetValue[string, string](ctx.Record, interceptor.ToolCall); ok {
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
	if len(data) < 5 {
		return
	}
	data = data[6:]
	if len(data) == 0 {
		return
	}

	var response model.Response
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		logger.Sugar().Errorf("解码失败： %v -- %s", err, data)
		return
	}

	choice := response.Choices[0]
	if choice.FinishReason != nil && *choice.FinishReason == "stop" {
		return true
	}

	chunk := choice.Delta.Content
	logger.Sugar().Debug("----- raw -----")
	logger.Sugar().Debug(chunk)

	chunk = interceptor.ExecuteInterceptors(ctx, chunk, false)
	for _, yield := range calls {
		yield()
	}
	if chunk == "" {
		return
	}

	channel <- &model.ChunkBodies{Chunk: chunk, Stream: true}
	return
}
