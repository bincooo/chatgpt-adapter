package plat

import (
	"context"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/claude-api"
	"strings"
	"time"
)

type ClaudeBot struct {
	sessions map[string]*claude.Chat
}

func NewClaudeBot() types.Bot {
	return &ClaudeBot{
		sessions: make(map[string]*claude.Chat, 0),
	}
}

func (bot *ClaudeBot) Reply(ctx types.ConversationContext) chan types.PartialResponse {
	message := make(chan types.PartialResponse)
	go func() {
		defer close(message)
		session, ok := bot.sessions[ctx.Id]
		if !ok {
			chat := claude.New(ctx.Token, ctx.AppId)
			if err := chat.NewChannel("chat-7890"); err != nil {
				message <- types.PartialResponse{Error: err}
				return
			}
			session = chat
			bot.sessions[ctx.Id] = session
		}

		timeout, cancel := context.WithTimeout(context.TODO(), 3*time.Minute)
		defer cancel()

		partialResponse, err := session.Reply(timeout, ctx.Prompt)
		if err != nil {
			message <- types.PartialResponse{Error: err}
			return
		}

		pos := 0
		r := CacheBuffer{
			H: func(self *CacheBuffer) error {
				response, ok := <-partialResponse
				if !ok {
					self.Closed = true
					return nil
				}

				if response.Error != nil {
					self.Closed = true
					return response.Error
				}

				// 截掉结尾的 Typing
				text := response.Text
				if strings.HasSuffix(text, claude.Typing) {
					text = strings.TrimSuffix(text, "\n\n"+claude.Typing)
					text = strings.TrimSuffix(text, claude.Typing)
				}

				str := []rune(text)
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
	}()
	return message
}

func (bot *ClaudeBot) Reset(id string) bool {
	delete(bot.sessions, id)
	return true
}
