package nexos

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/xllm-go/g/interceptor"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

func createChannel(ctx *model.Ctx, reader io.Reader) chan *model.ChunkBodies {
	channel := make(chan *model.ChunkBodies, 1)

	go func() {
		scanner := bufio.NewScanner(reader)
		chainInterceptor := interceptor.ExecuteInterceptors(ctx, channel)
		defer func() {
			chunk := chainInterceptor("", true)
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
				if ok := scan(scanner, channel, chainInterceptor); ok {
					return
				}
			}
		}
	}()

	return channel
}

func scan(scanner *bufio.Scanner, channel chan *model.ChunkBodies, chainInterceptor interceptor.ChainInterceptor) (ok bool) {
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

	chunk = chainInterceptor(chunk, false)
	if chunk == "" {
		return
	}

	channel <- &model.ChunkBodies{Chunk: chunk, Stream: true}
	return
}
