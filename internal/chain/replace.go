package chain

import (
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"strings"
	"time"
)

// 内置默认的替换标记符
type ReplaceInterceptor struct {
	types.BaseInterceptor
}

func (c *ReplaceInterceptor) Before(bot types.Bot, ctx *types.ConversationContext) (bool, error) {
	if ctx.Bot == vars.Bing {
		// 发你好也道歉？不愧是Bing
		if strings.Contains(ctx.Prompt, "嗨") {
			ctx.Prompt = strings.ReplaceAll(ctx.Prompt, "嗨", "Hi。（用中文回复）")
		} else if strings.Contains(ctx.Prompt, "你好") {
			ctx.Prompt = strings.ReplaceAll(ctx.Prompt, "你好", "Excuse me。（用中文回复）")
		}
	}
	if ctx.Format != "" {
		ctx.Prompt = replace(ctx.Format, ctx.Prompt)
	}
	ctx.Prompt = replace(ctx.Prompt, "")
	return true, nil
}

func replace(source string, content string) string {
	if content != "" {
		content = strings.ReplaceAll(content, "\"", "\\u0022")
		if strings.Contains(source, "[content]") {
			source = strings.Replace(source, "[content]", content, -1)
		} else {
			source = source + content
		}
	}

	if strings.Contains(source, "[date]") {
		date := time.Now().Format("2006-01-02 15:04:05")
		source = strings.Replace(source, "[date]", date, -1)
	}

	return source
}
