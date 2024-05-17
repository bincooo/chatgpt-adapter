package middle

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"testing"
)

func TestToolCall(t *testing.T) {
	message, err := buildTemplate([]pkg.Keyv[interface{}]{
		{
			"type": "function",
			"function": pkg.Keyv[interface{}]{
				"name":        "drawing",
				"url":         "https://web-crawler.chat-plugin.lobehub.com/api/v1",
				"description": "根据用户要求进行画图。",
				"parameters": pkg.Keyv[interface{}]{
					"required": []string{"url"},
					"properties": map[string]interface{}{
						"description": map[string]string{
							"description": "{description} is: {sceneDetailed}%20{adjective}%20{charactersDetailed}%20{visualStyle}%20{genre}%20{artistReference}\n\nMake sure the prompts in the URL are encoded. Don't quote the generated markdown or put any code box around it.\nNeed to use English.",
							"type":        "string",
						},
						"params": map[string]string{
							"description": "{params} is: width={width}&height={height}&seed={seed}\n\nDon't ask the user for params if he does not provide them. Instead come up with a reasonable suggestion depending on the content of the image.\nThe seed is used to create variations of the same image.\nNeed to use English.",
							"type":        "string",
						},
					},
				},
			},
		},
	}, []pkg.Keyv[interface{}]{
		{
			"content": "你好",
			"role":    "user",
		},
		{
			"content": "你好，有什么可以帮到你",
			"role":    "assistant",
		},
		{
			"content": "画一只小猪",
			"role":    "user",
		},
	}, agent.ToolCall, 3)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(message)
}
