// 该包下仅提供给iocgo工具使用的，不需要理会 Injects 的错误，在编译过程中生成
package scan

import (
	"github.com/iocgo/sdk"

	_ "chatgpt-adapter/relay/alloc/bing"
	_ "chatgpt-adapter/relay/alloc/coze"
	_ "chatgpt-adapter/relay/alloc/grok"
	_ "chatgpt-adapter/relay/alloc/you"

	"chatgpt-adapter/relay/hf"
	"chatgpt-adapter/relay/llm/bing"
	"chatgpt-adapter/relay/llm/blackbox"
	"chatgpt-adapter/relay/llm/coze"
	"chatgpt-adapter/relay/llm/cursor"
	"chatgpt-adapter/relay/llm/deepseek"
	"chatgpt-adapter/relay/llm/grok"
	"chatgpt-adapter/relay/llm/lmsys"
	"chatgpt-adapter/relay/llm/qodo"
	"chatgpt-adapter/relay/llm/v1"
	"chatgpt-adapter/relay/llm/windsurf"
	"chatgpt-adapter/relay/llm/you"
	"chatgpt-adapter/relay/llm/zed"
	"chatgpt-adapter/relay/pg"
)

func Injects(container *sdk.Container) (err error) {
	err = v1.Injects(container)
	if err != nil {
		return
	}

	err = bing.Injects(container)
	if err != nil {
		return
	}

	err = coze.Injects(container)
	if err != nil {
		return
	}

	err = cursor.Injects(container)
	if err != nil {
		return
	}

	err = windsurf.Injects(container)
	if err != nil {
		return
	}

	err = you.Injects(container)
	if err != nil {
		return
	}

	err = deepseek.Injects(container)
	if err != nil {
		return
	}

	err = grok.Injects(container)
	if err != nil {
		return
	}

	err = lmsys.Injects(container)
	if err != nil {
		return
	}

	err = qodo.Injects(container)
	if err != nil {
		return
	}

	err = pg.Injects(container)
	if err != nil {
		return
	}

	err = hf.Injects(container)
	if err != nil {
		return
	}

	err = blackbox.Injects(container)
	if err != nil {
		return
	}

	err = zed.Injects(container)
	if err != nil {
		return
	}

	err = rejects(container)
	if err != nil {
		return
	}

	return
}
