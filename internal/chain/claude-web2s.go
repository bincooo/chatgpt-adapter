package chain

import (
	"github.com/bincooo/MiaoX/internal/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"strings"
)

type ClaudeWeb2sInterceptor struct {
	types.BaseInterceptor
}

func (c *ClaudeWeb2sInterceptor) Before(bot types.Bot, ctx *types.ConversationContext) bool {
	if ctx.Bot == vars.Claude && ctx.Model == vars.Model4WebClaude2S {
		const (
			Assistant = "A:"
			Human     = "H:"
		)
		if !strings.Contains(ctx.Prompt, "[history]") {
			return true
		}

		messages := store.GetMessages(ctx.Id)
		if len(messages) == 0 {
			return true
		}

		history := ""
		for _, message := range messages {
			if message["author"] == "bot" {
				text := strings.TrimSpace(message["text"])
				text = strings.ReplaceAll(text, "\n\n", "\n")
				if !strings.HasPrefix(text, Assistant) {
					history += Assistant + " " + text
				} else {
					history += text
				}
			}

			if message["author"] == "user" {
				text := strings.TrimSpace(message["text"])
				if !strings.HasPrefix(text, Human) {
					history += Human + " " + text
				} else {
					history += text
				}
			}

			history += "\n\n"
		}

		ctx.Prompt = strings.Replace(ctx.Prompt, "[history]", history, -1)
	}
	return true
}
