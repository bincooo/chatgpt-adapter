package plat

import (
	"context"
	"github.com/bincooo/MiaoX/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"github.com/bincooo/edge-api"
	"github.com/sirupsen/logrus"
	"strings"
)

type BingBot struct {
	sessions map[string]*edge.Chat
}

func NewBingBot() types.Bot {
	return &BingBot{
		make(map[string]*edge.Chat),
	}
}

func (bot *BingBot) Reply(ctx types.ConversationContext) chan types.PartialResponse {
	var message = make(chan types.PartialResponse)
	go func() {
		defer close(message)
		session, ok := bot.sessions[ctx.Id]
		if !ok {
			chat, err := edge.New(ctx.Token, ctx.BaseURL)
			chat.Model = ctx.Model
			chat.TraceId = ctx.AppId
			if err != nil {
				message <- types.PartialResponse{Error: err}
				return
			}
			session = chat
			bot.sessions[ctx.Id] = session
		}

		timeout, cancel := context.WithTimeout(context.TODO(), Timeout)
		defer cancel()
		messages := store.GetMessages(ctx.Id)
		if ctx.Preset != "" {
			messages = append([]map[string]string{
				{
					"author": "user",
					"text":   ctx.Preset,
				},
				{
					"author": "bot",
					"text":   "明白了，有什么可以帮助你的？",
				},
			}, messages...)
		}
		partialResponse, err := session.Reply(timeout, ctx.Prompt, messages)
		if err != nil {
			message <- types.PartialResponse{Error: err}
			return
		}

		logrus.Info("[MiaoX] - Bot.Session: ", session.Session.ConversationId)
		bot.handle(ctx, partialResponse, message)
	}()
	return message
}

func (bot *BingBot) Remove(id string) bool {
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

func (bot *BingBot) handle(ctx types.ConversationContext, partialResponse chan edge.PartialResponse, message chan types.PartialResponse) {
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
					logrus.Error(response.Error)
				}

				if response.Type == 2 {
					if response.Item.Throttling != nil {
						vars.BingMaxMessage = response.Item.Throttling.Max
					}

					messages := response.Item.Messages
					if messages == nil {
						goto label
					}

					for _, value := range *messages {
						if value.Type == "Disengaged" {
							delete(bot.sessions, ctx.Id)
							if response.Text == "" {
								response.Text = "对不起，我不想继续这个对话。我还在学习中，所以感谢你的理解和耐心。🙏"
							}
						}
					}

				label:
				}

				str := []rune(response.Text)
				length := len(str)
				if pos >= length {
					return nil
				}
				self.Cache += string(str[pos:])
				pos = length
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

// =======
