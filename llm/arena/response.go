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
		scanner := bufio.NewScanner(reader)
		matchers := model.JustValue[string, []model.Matcher](ctx.Record, model.Matchers)

		defer func() {
			if chunk := model.ExecMatchers(matchers, "", true); chunk != "" {
				channel <- model.CreateChunk(chunk, "")
			}
			close(channel)
		}()

		for {
			if ok := scan(scanner, matchers, channel); ok {
				break
			}
		}
	}()

	return channel
}

func scan(scanner *bufio.Scanner, matchers []model.Matcher, channel chan *model.ChunkBodies) (ok bool) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			logger.Sugar().Error(err)
			channel <- &model.ChunkBodies{
				Chunk: "error: " + err.Error(),
			}
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

	chunk = model.ExecMatchers(matchers, chunk, false)
	model.SplitEach(chunk, func(message string) {
		channel <- &model.ChunkBodies{
			Chunk: message,
		}
	})
	return
}
