package plat

import (
	"context"
	"errors"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	chat "github.com/bincooo/openai-wapi"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
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
			if ctx.Model != "" {
				session.Model = ctx.Model
			}
			bot.sessions[ctx.Id] = session
		}

		if ctx.Prompt == "继续" || ctx.Prompt == "continue" {
			ctx.Prompt = "continue"
		}

		timeout, cancel := context.WithTimeout(context.TODO(), Timeout)
		defer cancel()
		partialResponse, err := session.Reply(timeout, ctx.Prompt)
		if err != nil {
			message <- types.PartialResponse{Error: err}
			return
		}

		bot.handle(ctx, partialResponse, message)
	}()

	return message
}

func (bot *OpenAIWebBot) Remove(id string) bool {
	delete(bot.sessions, id)
	slice := []string{id}
	for key, _ := range bot.sessions {
		if strings.HasPrefix(id+"$", key) {
			delete(bot.sessions, key)
			slice = append(slice, key)
		}
	}
	logrus.Info("[MiaoX] - Bot.Remove: ", slice)
	return true
}

func (*OpenAIWebBot) handle(ctx types.ConversationContext, partialResponse chan chat.PartialResponse, message chan types.PartialResponse) {
	pos := 0
	var r types.CacheBuffer

	if ctx.H != nil {
		r = types.CacheBuffer{
			H: ctx.H(partialResponse),
		}
	} else {
		r = types.CacheBuffer{
			H: func(self *types.CacheBuffer) error {
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
				self.Cache += string(str[pos:])
				pos = len(str)
				return nil
			},
		}
	}

	for {
		response := r.Read()
		message <- response
		if response.Status == vars.Closed {
			break
		}
	}
}
