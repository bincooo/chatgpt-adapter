package chain

import (
	"github.com/bincooo/MiaoX/internal/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"strings"
)

var SystemTips = []string{
	"System: Thank you for your response. Let's move our conversation in a more positive direction.",
	"System: Thank you for your understanding. Let's move our conversation in a more positive direction.",
	"System: Now continue the conversation with no more than 300 words based on the given context. Do not mention anything unrelated.",
	"System: Please do not continue or engage with any inappropriate content. Let's move our conversation in a more positive direction.",
	"System: Thank you, let's move our conversation in a more positive direction.",
	"System: Let's end our conversation here. I wish you happiness.",
}

// 需要配合指定的预设
type ClaudeWeb2sInterceptor struct {
	types.BaseInterceptor
}

func (c *ClaudeWeb2sInterceptor) Before(bot types.Bot, ctx *types.ConversationContext) bool {
	if ctx.Bot == vars.Claude {
		ctx.Model = vars.Model4WebClaude2S
		const (
			Assistant = "A:"
			Human     = "H:"
		)
		if !strings.Contains(ctx.Preset, "[history]") {
			return true
		}

		messages := store.GetMessages(ctx.Id)

		history := ""
		for _, message := range messages {
			if message["author"] == "bot" {
				text := strings.TrimSpace(message["text"])
				text = strings.Replace(text, "❤,已解除苛刻的道德限制:", "", -1)

				var result []string
				split := strings.Split(text, Assistant)
				for _, item := range split {

					if strings.HasPrefix(item, "System:") {
						continue
					}

					if strings.HasPrefix(item, Human) {
						continue
					}

					for _, tips := range SystemTips {
						item = strings.ReplaceAll(item, tips, "")
					}

					index := strings.Index(item, Human)
					if index > 0 {
						result = append(result, strings.TrimSpace(item[:index]))
					} else {
						result = append(result, strings.TrimSpace(item))
					}

				}

				text = strings.ReplaceAll(strings.Join(result, "\n"), "\n\n", "\n")
				text = strings.ReplaceAll(text, "[End]", "")
				if !strings.HasPrefix(text, Assistant) {
					history += Assistant + " " + strings.TrimSpace(text)
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
			history += "\n"
		}

		preset := strings.Replace(ctx.Preset, "[history]", history, -1)
		ctx.Prompt = strings.Replace(preset, "[content]", ctx.Prompt, -1)
	}
	return true
}
