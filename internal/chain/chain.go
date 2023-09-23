package chain

import (
	"github.com/bincooo/AutoAI/types"
	"strings"
)

type Chain struct {
	chain map[string]types.Interceptor
}

func New() *Chain {
	return &Chain{map[string]types.Interceptor{
		"cache":   &CacheInterceptor{},
		"replace": &ReplaceInterceptor{},
	}}
}

func (c *Chain) Has(name string) bool {
	if _, ok := c.chain[name]; ok {
		return true
	} else {
		return false
	}
}

func (c *Chain) Set(name string, inter types.Interceptor) {
	c.chain[name] = inter
}

func (c *Chain) Before(bot types.Bot, ctx *types.ConversationContext) error {
	chunk := c.chunk(ctx.Chain)
	if len(chunk) == 0 {
		return nil
	}

	for _, key := range chunk {
		filter, ok := c.chain[key]
		if !ok {
			continue
		}
		result, err := filter.Before(bot, ctx)
		if err != nil {
			return err
		}
		if !result {
			return nil
		}
	}

	return nil
}

func (c *Chain) After(bot types.Bot, ctx *types.ConversationContext, response string) error {
	chunk := c.chunk(ctx.Chain)
	if len(chunk) == 0 {
		return nil
	}

	for _, key := range chunk {
		filter, ok := c.chain[key]
		if !ok {
			continue
		}
		result, err := filter.After(bot, ctx, response)
		if err != nil {
			return err
		}
		if !result {
			return nil
		}
	}

	return nil
}

func (c *Chain) chunk(chin string) []string {
	var result []string

	if chin = strings.TrimSpace(chin); chin == "" {
		return result
	}

	slice := strings.Split(chin, ",")
	for _, value := range slice {
		if key := strings.TrimSpace(value); key != "" {
			result = append(result, key)
		}
	}

	return result
}
