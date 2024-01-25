package bing

import (
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"testing"
)

func Test(t *testing.T) {
	toolsMap, prompt, err := buildToolsPrompt([]struct {
		Fun gpt.Function `json:"function"`
		T   string       `json:"type"`
	}{
		{
			T: "function",
			Fun: gpt.Function{
				Name:        "website-crawler____getWebsiteContent",
				Url:         "https://web-crawler.chat-plugin.lobehub.com/api/v1",
				Description: "提取网页内容",
				Params: struct {
					Properties map[string]interface{} `json:"properties"`
					Required   []string               `json:"required"`
				}{
					Required: []string{"url"},
					Properties: map[string]interface{}{
						"url": map[string]string{
							"type":        "string",
							"description": "网页链接",
						},
					},
				},
			},
		},
	}, []map[string]string{
		{
			"content": "你好",
			"role":    "user",
		},
		{
			"content": "你好，有什么可以帮到你",
			"role":    "assistant",
		},
		{
			"content": "https://juejin.cn/post/7229480315353514045 阅读这个链接的内容",
			"role":    "user",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(toolsMap)
	t.Log(prompt)
}
