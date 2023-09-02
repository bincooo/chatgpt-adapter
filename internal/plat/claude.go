package plat

import (
	"context"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"github.com/bincooo/claude-api"
	clTypes "github.com/bincooo/claude-api/types"
	clVars "github.com/bincooo/claude-api/vars"
	"os"
	"strings"
)

const (
	ClackTyping = "_Typing…_"
)

var (
	deleteHistory = loadEnvBool("DELETE_HISTORY", false)
)

func loadEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	return strings.TrimSpace(strings.ToLower(value)) == "true"
}

type ClaudeBot struct {
	sessions map[string]clTypes.Chat
}

func NewClaudeBot() types.Bot {
	return &ClaudeBot{
		sessions: make(map[string]clTypes.Chat, 0),
	}
}

func (bot *ClaudeBot) Reply(ctx types.ConversationContext) chan types.PartialResponse {
	message := make(chan types.PartialResponse)
	go func() {
		defer close(message)
		session, ok := bot.sessions[ctx.Id]
		if !ok {
			model := ctx.Model
			if model == vars.Model4WebClaude2S {
				model = clVars.Model4WebClaude2
			}
			options := claude.NewDefaultOptions(ctx.Token, ctx.AppId, model)
			options.BaseURL = ctx.BaseURL
			options.Agency = ctx.Proxy
			chat, err := claude.New(options)
			if err != nil {
				message <- types.PartialResponse{Error: err}
				return
			}
			if ctx.Model == clVars.Model4Slack {
				if err = chat.NewChannel("chat-7890"); err != nil {
					message <- types.PartialResponse{Error: err}
					return
				}
			}
			session = chat
			bot.sessions[ctx.Id] = session
		}

		timeout, cancel := context.WithTimeout(context.TODO(), Timeout)
		defer cancel()

		var attrs []clTypes.Attachment = nil
		var prompt = ctx.Prompt

		// S模式，每次创建一个新会话，使用附件方式对话
		if ctx.Model == vars.Model4WebClaude2S {
			prompt = ""
			attrs = []clTypes.Attachment{
				{
					Content:  ctx.Prompt,
					FileName: "paste.txt",
					FileSize: len(ctx.Prompt),
					FileType: "txt",
				},
			}
			defer bot.Remove(ctx.Id)
		}
		partialResponse, err := session.Reply(timeout, prompt, attrs)
		if err != nil {
			message <- types.PartialResponse{Error: err}
			return
		}

		var r types.CacheBuffer

		if ctx.H != nil {
			r = types.CacheBuffer{
				H: ctx.H(partialResponse),
			}
		} else {
			pos := 0
			r = types.CacheBuffer{
				H: func(self *types.CacheBuffer) error {
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
					if ctx.Model == clVars.Model4Slack && strings.HasSuffix(text, ClackTyping) {
						text = strings.TrimSuffix(text, "\n\n"+ClackTyping)
						text = strings.TrimSuffix(text, ClackTyping)
					}

					if ctx.Model == clVars.Model4Slack {
						str := []rune(text)
						self.Cache += string(str[pos:])
						pos = len(str)
						return nil
					} else {
						self.Cache += text
						return nil
					}
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
	}()
	return message
}

func (bot *ClaudeBot) Remove(id string) bool {
	if session, ok := bot.sessions[id]; ok {
		if deleteHistory {
			go session.Delete()
		}
		delete(bot.sessions, id)
	}
	for key, se := range bot.sessions {
		if strings.HasPrefix(id+"$", key) {
			if deleteHistory {
				go se.Delete()
			}
			delete(bot.sessions, key)
		}
	}
	return true
}
