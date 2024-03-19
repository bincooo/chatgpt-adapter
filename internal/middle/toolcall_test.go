package middle

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"testing"
)

func TestCQConditions(t *testing.T) {
	prompt, err := BuildToolCallsTemplate([]struct {
		Fun gpt.Function `json:"function"`
		T   string       `json:"type"`
	}{
		{
			T: "function",
			Fun: gpt.Function{
				Name:        "drawing",
				Url:         "https://web-crawler.chat-plugin.lobehub.com/api/v1",
				Description: "根据用户要求进行画图。",
				Params: struct {
					Properties map[string]interface{} `json:"properties"`
					Required   []string               `json:"required"`
					Type       string                 `json:"type"`
				}{
					Required: []string{"url"},
					Properties: map[string]interface{}{
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
			"content": "画一只小猪",
			"role":    "user",
		},
	}, agent.CQConditions, 3)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(prompt)
}

func TestExtractJson(t *testing.T) {
	prompt, err := BuildToolCallsTemplate([]struct {
		Fun gpt.Function `json:"function"`
		T   string       `json:"type"`
	}{
		{
			T: "function",
			Fun: gpt.Function{
				Name:        "drawing",
				Url:         "https://web-crawler.chat-plugin.lobehub.com/api/v1",
				Description: "根据用户要求进行画图。",
				Params: struct {
					Properties map[string]interface{} `json:"properties"`
					Required   []string               `json:"required"`
					Type       string                 `json:"type"`
				}{
					Required: []string{"url"},
					Properties: map[string]interface{}{
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
			"content": "画一只小猪",
			"role":    "user",
		},
	}, agent.ExtractJson, 3)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(prompt)
}
