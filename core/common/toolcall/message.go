package toolcall

import (
	"chatgpt-adapter/core/gin/model"
)

func ExtractToolMessages(completion *model.Completion) (toolMessages []model.Keyv[interface{}]) {
	for i := len(completion.Messages) - 1; i >= 0; i-- {
		message := completion.Messages[i]
		if message.Is("role", "tool") || (message.Is("role", "assistant") && message.Has("tool_calls")) {
			toolMessages = append(toolMessages, message)
			continue
		}

		completion.Messages = completion.Messages[:i+1]
		break
	}
	return
}
