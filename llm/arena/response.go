package arena

import (
	"bufio"
	"io"
	"strconv"
	"strings"

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
		think = bodies.Think
	}
	return model.CreateChunk(chunk, think)
}

func createChannel(ctx *model.Ctx, reader io.Reader) chan *model.ChunkBodies {
	channel := make(chan *model.ChunkBodies)

	go func() {
		defer close(channel)
		scanner := bufio.NewScanner(reader)
		for {
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					logger.Sugar().Error(err)
				}
				break
			}

			data := scanner.Text()
			logger.Sugar().Debugf("%s", data)
			if len(data) < 3 {
				continue
			}

			state := data[:2]
			data = data[3:]
			if state == "ad" && strings.Contains(data, "\"finishReason\":\"stop\"") {
				break
			}

			if state == "a2" {
				continue
			}

			chunk, err := strconv.Unquote(data)
			if err != nil {
				logger.Sugar().Errorf("转义失败： %v", err)
				continue
			}

			logger.Sugar().Debug("----- raw -----")
			logger.Sugar().Debug(chunk)

			if len(chunk) == 0 {
				continue
			}

			channel <- &model.ChunkBodies{
				Chunk: chunk,
			}
		}
	}()

	return channel
}
