package bing

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
)

func buildToolsPrompt(
	tools []struct {
		Fun gpt.Function `json:"function"`
		T   string       `json:"type"`
	},
	messages []map[string]string,
) (toolsMap map[string]string, prompt string, err error) {
	pMessages, content, err := buildConversation(messages)
	if err != nil {
		return nil, "", err
	}

	build := middle.NewTemplateWrapper().
		Variables("tools", tools).
		Variables("pMessages", pMessages).
		Variables("content", content).
		Func("rand", middle.RandString).
		Func("contains", func(s1 []string, s2 string) bool {
			return middle.Contains(s1, s2)
		}).
		Func("setId", func(index int, value string) string {
			tools[index].Fun.Id = value
			return ""
		}).
		Func("inc", func(i, s int) int {
			return i + s
		}).
		Build()
	prompt, err = build(agent.BingToolCallsTemplate)
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
