package chain

import (
	"github.com/bincooo/MiaoX/internal/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"strings"
)

// continue指令，尽可能保证bing能输出完整内容而不被截断
type BingInterceptor struct {
	types.BaseInterceptor
}

var (
	substrSlice = []string{
		"对不起，我不想再和你说话",
		"Unfortunately, we need to move on!",
		"Sorry, I think we need to move on!",
		"Oops, I think we've reached the end of this conversation.",
		"Sorry, it looks like I need to chat about something else.",
	}
)

func (c *BingInterceptor) Before(bot types.Bot, ctx *types.ConversationContext) bool {
	messages := store.GetMessages(ctx.Id)
	if len(messages) == 0 {
		return true
	}

	for idx := len(messages) - 1; idx > 0; idx-- {
		if _, ok := messages[idx]["_"]; ok {
			messages = append(messages[:idx], messages[idx+1:]...)
		}
	}

	if len(messages) == 0 {
		return true
	}

	message := messages[len(messages)-1]["text"]
	handle := func(str string) {
		message = message[:strings.Index(message, str)]
		if len(message) == 0 {
			messages = messages[:len(messages)-2]
		} else {
			messages = append(messages[:len(messages)-1], map[string]string{
				"author": "bot",
				"text":   message[:len(message)-5],
			})
		}
	}

	for _, str := range substrSlice {
		if strings.Contains(message, str) {
			handle(str)
			break
		}
	}

	if strings.Contains(ctx.Prompt, "continue") {
		substr := message
		if len(substr) > 10 {
			substr = message[:len(message)-5]
		}

		messages = append(messages[:len(messages)-1], map[string]string{
			"author": "bot",
			"text":   substr,
		})
		// 这里多次插入相同对话，是为了强调回复
		messages = append(messages, map[string]string{
			"author": "user",
			"text":   ctx.Prompt,
			"_":      "1",
		})
		messages = append(messages, map[string]string{
			"author": "bot",
			"text":   substr,
			"_":      "1",
		})
		messages = append(messages, map[string]string{
			"author": "user",
			"text":   ctx.Prompt,
			"_":      "1",
		})
		messages = append(messages, map[string]string{
			"author": "bot",
			"text":   message,
			"_":      "1",
		})

		maxMessages := 30
		if ctx.Bot == vars.Bing {
			maxMessages = vars.BingMaxMessage - 1
			if ctx.Preset != "" {
				maxMessages--
			}
		}

		if len(messages) > maxMessages*2 {
			messages = messages[len(messages)-maxMessages*2:]
		}
	}
	store.CacheMessages(ctx.Id, messages)
	return true
}
