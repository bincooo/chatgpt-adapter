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
	cache map[string]string
}

func (c *CacheInterceptor) Before(bot types.Bot, ctx *types.ConversationContext) bool {
	if c.cache == nil {
		c.cache = make(map[string]string)
	}
	c.cache[ctx.Id] = ctx.Prompt
	return true
}

func (c *CacheInterceptor) After(bot types.Bot, ctx *types.ConversationContext, response string) bool {
	c.cacheAfter(ctx, response)
	return true
}

func (c *CacheInterceptor) cacheAfter(ctx *types.ConversationContext, response string) {
	defer delete(c.cache, ctx.Id)
	if hasIg(ctx.Bot, response) {
		return
	}

	messages := store.GetMessages(ctx.Id)
	if response != "" {
		prompt := c.cache[ctx.Id]
		messages = append(messages, map[string]string{
			"author": "user",
			"text":   prompt,
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
