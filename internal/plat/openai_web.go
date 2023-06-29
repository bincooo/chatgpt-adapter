package plat

import (
	"context"
	"errors"
	"github.com/bincooo/MiaoX/types"
	chat "github.com/bincooo/openai-wapi"
	"github.com/sirupsen/logrus"
	"io"
	"time"
)

type OpenAIWebBot struct {
	sessions map[string]*chat.Chat
}

func NewOpenAIWebBot() types.Bot {
	return &OpenAIWebBot{
		sessions: map[string]*chat.Chat{},
	}
}

func (bot *OpenAIWebBot) Reply(ctx types.ConversationContext) chan types.PartialResponse {
	var message = make(chan types.PartialResponse)
	go func() {
		defer close(message)

		session, ok := bot.sessions[ctx.Id]
		if !ok {
			session = chat.New(ctx.Token, ctx.BaseURL)
			bot.sessions[ctx.Id] = session
		}

		if ctx.Prompt == "继续" || ctx.Prompt == "continue" {
			ctx.Prompt = "continue"
		}

		timeout, cancel := context.WithTimeout(context.TODO(), 3*time.Minute)
		defer cancel()
		partialResponse, err := session.Reply(timeout, ctx.Prompt)
		if err != nil {
			message <- types.PartialResponse{Error: err}
			return
		}

		bot.handle(partialResponse, message)
	}()

	return message
}

func (bot *OpenAIWebBot) Reset(key string) bool {
	delete(bot.sessions, key)
	return true
}

func (*OpenAIWebBot) handle(partialResponse chan chat.PartialResponse, message chan types.PartialResponse) {
	pos := 0
	r := CacheBuffer{
		H: func(self *CacheBuffer) error {
			response, ok := <-partialResponse
			if !ok {
				self.Closed = true
				return nil
			}

			if response.Error != nil {
				if response.Error == io.EOF {
					self.Closed = true
					return nil
				}
				return response.Error
			}

			if response.ResponseError != nil {
				logrus.Error(response.ResponseError)
				return errors.New("error")
			}

			str := []rune(response.Message.Content.Parts[0])
			self.cache += string(str[pos:])
			pos = len(str)
			return nil
		},
	}

	for {
		response := r.Read()
		message <- response
		if response.Closed {
			break
		}
	}
}
