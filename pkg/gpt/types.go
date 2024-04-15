package gpt

type ChatCompletionRequest struct {
	Messages      []map[string]string `json:"messages"`
	Model         string              `json:"model"`
	MaxTokens     int                 `json:"max_tokens"`
	StopSequences []string            `json:"stop_sequences"`
	Temperature   float32             `json:"temperature"`
	TopK          int                 `json:"topK"`
	TopP          float32             `json:"topP"`
	Stream        bool                `json:"stream"`
	Tools         []struct {
		Fun Function `json:"function"`
		T   string   `json:"type"`
	} `json:"tools"`
	ToolChoice string `json:"tool_choice"`
}

type ChatGenerationRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
	Style  string `json:"style"`
}

type Function struct {
	Id          string `json:"-"`
	Name        string `json:"name"`
	Url         string `json:"url"`
	Description string `json:"description"`
	Params      struct {
		Properties map[string]interface{} `json:"properties"`
		Required   []string               `json:"required"`
		Type       string                 `json:"type"`
	} `json:"parameters"`
}

type ChatCompletionResponse struct {
	Id      string                         `json:"id"`
	Object  string                         `json:"object"`
	Created int64                          `json:"created"`
	Model   string                         `json:"model"`
	Choices []ChatCompletionResponseChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Usage map[string]int `json:"usage"`
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
	FinishReason *string `json:"finish_reason"`
}
