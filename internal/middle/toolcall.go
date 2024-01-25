package middle

import (
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
)

func BuildToolCallsTemplate(
	tools []struct {
		Fun gpt.Function `json:"function"`
		T   string       `json:"type"`
	},
	messages []map[string]string,
	toolCallsTemplate string,
	maxMessage int,
) (toolsMap map[string]string, prompt string, err error) {
	pMessages := messages
	content := "continue"
	if messageL := len(messages); messageL > 0 && messages[messageL-1]["role"] == "user" {
		content = messages[messageL-1]["content"]
		if maxMessage == 0 {
			pMessages = make([]map[string]string, 0)
		} else if maxMessage > 0 && messageL > maxMessage {
			pMessages = messages[messageL-maxMessage : messageL-1]
		} else {
			pMessages = messages[:messageL-1]
		}
	}

	build := NewTemplateWrapper().
		Variables("tools", tools).
		Variables("pMessages", pMessages).
		Variables("content", content).
		Func("rand", RandString).
		Func("contains", func(s1 []string, s2 string) bool {
			return Contains(s1, s2)
		}).
		Func("setId", func(index int, value string) string {
			tools[index].Fun.Id = value
			return ""
		}).
		Func("inc", func(i, s int) int {
			return i + s
		}).
		Build()
	prompt, err = build(toolCallsTemplate)
	if err != nil {
		return
	}

	toolsMap = make(map[string]string)
	for _, tool := range tools {
		if tool.T == "function" {
			f := tool.Fun
			toolsMap[f.Id] = f.Name
		}
	}

	return
}
