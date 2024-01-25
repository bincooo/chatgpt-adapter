package gpt

type ChatCompletionRequest struct {
	Messages      []map[string]string `json:"messages"`
	Model         string              `json:"model"`
	StopSequences []string            `json:"stop_sequences"`
	Temperature   float32             `json:"temperature"`
	Stream        bool                `json:"stream"`
	Tools         []struct {
		Fun Function `json:"function"`
		T   string   `json:"type"`
	} `json:"tools"`
}

type Function struct {
	Id          string `json:"-"`
	Name        string `json:"name"`
	Url         string `json:"url"`
	Description string `json:"description"`
	Params      struct {
		Properties map[string]interface{} `json:"properties"`
		Required   []string               `json:"required"`
	} `json:"parameters"`
}

type ChatCompletionResponse struct {
	Id      string                         `json:"id"`
	Object  string                         `json:"object"`
	Created int64                          `json:"created"`
	Model   string                         `json:"model"`
	Choices []ChatCompletionResponseChoice `json:"choices"`
}

type ChatCompletionResponseChoice struct {
	Index   int `json:"index"`
	Message *struct {
		Role      string                   `json:"role"`
		Content   string                   `json:"content"`
		ToolCalls []map[string]interface{} `json:"tool_calls"`
	} `json:"message"`
	Delta *struct {
		Role      string                   `json:"role"`
		Content   string                   `json:"content"`
		ToolCalls []map[string]interface{} `json:"tool_calls"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}
