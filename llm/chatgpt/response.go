package chatgpt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/xllm-go/g/interceptor"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

type originResponse struct {
	Message struct {
		Id     string `json:"id"`
		Author struct {
			Role     string      `json:"role"`
			Name     interface{} `json:"name"`
			Metadata struct {
			} `json:"metadata"`
		} `json:"author"`
		CreateTime interface{} `json:"create_time"`
		UpdateTime interface{} `json:"update_time"`
		Content    struct {
			ContentType string   `json:"content_type"`
			Parts       []string `json:"parts"`
		} `json:"content"`
		Status   string  `json:"status"`
		EndTurn  bool    `json:"end_turn"`
		Weight   float64 `json:"weight"`
		Metadata struct {
			ParentId                         string        `json:"parent_id"`
			RequestId                        string        `json:"request_id"`
			EndTurn                          bool          `json:"end_turn"`
			ModelSwitcherDeny                []interface{} `json:"model_switcher_deny"`
			IsVisuallyHiddenFromConversation bool          `json:"is_visually_hidden_from_conversation"`
		} `json:"metadata"`
		Recipient string      `json:"recipient"`
		Channel   interface{} `json:"channel"`
	} `json:"message"`
	ConversationId string      `json:"conversation_id"`
	Error          interface{} `json:"error"`
	ErrorCode      interface{} `json:"error_code"`
}

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

		progress := scan(scanner, channel, chainInterceptor)
		for {
			select {
			case <-ctx.Context().Done():
				return
			default:
				if ok := progress(); ok {
					return
				}
			}
		}
	}()

	return channel
}

func scan(scanner *bufio.Scanner, channel chan *model.ChunkBodies, chainInterceptor interceptor.ChainInterceptor) func() (ok bool) {
	progress := ""
	pos := 0
	return func() (ok bool) {
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

		if data == "[DONE]" {
			return true
		}

		var response originResponse
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			logger.Sugar().Errorf("解码失败： %v -- %s", err, data)
			return
		}

		choice := response.Message
		if choice.Author.Role != "assistant" {
			return
		}

		if choice.Status == "in_progress" {
			progress = response.Message.Id
		}

		if progress == response.Message.Id && choice.Status == "finished_successfully" && choice.Metadata.EndTurn {
			return true
		}

		if progress != response.Message.Id || choice.Content.ContentType != "text" {
			return
		}

		chunk := choice.Content.Parts[0]
		if len(chunk) == 0 || len(chunk) < pos {
			return
		}

		chunk = chunk[pos:]
		pos = len(choice.Content.Parts[0])
		logger.Sugar().Debug("----- raw -----")
		logger.Sugar().Debug(chunk)

		chunk = chainInterceptor(chunk, false)
		if chunk == "" {
			return
		}

		channel <- &model.ChunkBodies{Chunk: chunk, Stream: true}
		return
	}
}
