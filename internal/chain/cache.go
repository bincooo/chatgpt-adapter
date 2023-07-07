package chain

import (
	"github.com/bincooo/MiaoX/internal/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"strings"
)

var (
	igMessages = []string{
		"我现在无法回答这个问题。",
		"我现在无法对此做出回应。",
		"对于这个问题很抱歉，让我们试试不同的话题",
	}
)

// 缓存对话记录
type CacheInterceptor struct {
	types.BaseInterceptor
}

func (c *CacheInterceptor) After(bot *types.Bot, ctx *types.ConversationContext, response string) bool {
	c.cachePreviousMessages(ctx, response)
	return true
}

func (c *CacheInterceptor) cachePreviousMessages(ctx *types.ConversationContext, response string) {
	if hasIg(ctx.Bot, response) {
		return
	}
	if response != "" {
		messages := store.GetMessages(ctx.Id)
		messages = append(messages, map[string]string{
			"author": "user",
			"text":   ctx.Prompt,
		})
		messages = append(messages, map[string]string{
			"author": "bot",
			"text":   response,
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
		store.CacheMessages(ctx.Id, messages)
	}
}

func hasIg(bot string, response string) bool {
	if bot != vars.Bing {
		return false
	}
	for _, value := range igMessages {
		if strings.Contains(response, value) {
			return true
		}
	}
	return false
}
