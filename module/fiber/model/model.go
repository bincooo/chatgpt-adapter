package model

type ModelEntity struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	By      string `json:"owned_by"`
}

type CompletionEntity struct {
	System        string                `json:"system,omitempty"`
	Messages      []Record[string, any] `json:"messages"`
	Tools         []Record[string, any] `json:"tools,omitempty"`
	Model         string                `json:"model,omitempty"`
	MaxTokens     int                   `json:"max_tokens"`
	StopSequences []string              `json:"stop,omitempty"`
	Temperature   float32               `json:"temperature"`
	TopK          int                   `json:"top_k,omitempty"`
	TopP          float32               `json:"top_p,omitempty"`
	Stream        bool                  `json:"stream,omitempty"`
	ToolChoice    interface{}           `json:"tool_choice,omitempty"`
}

type GenerationEntity struct {
	Model   string `json:"model"`
	Message string `json:"prompt"`
	N       int    `json:"n"`
	Size    string `json:"size"`
	Style   string `json:"style"`
	Quality string `json:"quality"`
}

type EmbeddingEntity struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
	Dimensions     int         `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
}

type ResponseEntity struct {
	Id      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []ChoiceEntity `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
	Usage map[string]interface{} `json:"usage,omitempty"`
}

type ChoiceEntity struct {
	Index   int `json:"index"`
	Message *struct {
		Role             string `json:"role,omitempty"`
		Content          string `json:"content,omitempty"`
		ReasoningContent string `json:"reasoning_content,omitempty"`

		ToolCalls []Record[string, any] `json:"tool_calls,omitempty"`
	} `json:"message,omitempty"`
	Delta *struct {
		Type             string `json:"type,omitempty"`
		Role             string `json:"role,omitempty"`
		Content          string `json:"content,omitempty"`
		ReasoningContent string `json:"reasoning_content,omitempty"`

		ToolCalls []Record[string, any] `json:"tool_calls,omitempty"`
	} `json:"delta,omitempty"`
	FinishReason *string `json:"finish_reason"`
}
