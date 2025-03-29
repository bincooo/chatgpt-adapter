//go:build 3rd

package scan

import (
	"chatgpt-adapter/relay/llm/trae"
)

func rejects(container *sdk.Container) (err error) {
	err = trae.Injects(container)
	if err != nil {
		return
	}
	return
}
