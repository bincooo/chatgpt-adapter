package model

type Model struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	By      string `json:"owned_by"`
}

type Completion struct {
	System        string              `json:"system,omitempty"`
	Messages      []Keyv[interface{}] `json:"messages"`
	Tools         []Keyv[interface{}] `json:"tools,omitempty"`
	Model         string              `json:"model,omitempty"`
	MaxTokens     int                 `json:"max_tokens"`
	StopSequences []string            `json:"stop,omitempty"`
	Temperature   float32             `json:"temperature"`
	TopK          int                 `json:"top_k,omitempty"`
	TopP          float32             `json:"top_p,omitempty"`
	Stream        bool                `json:"stream,omitempty"`
	ToolChoice    interface{}         `json:"tool_choice,omitempty"`
}

type Generation struct {
	Model   string `json:"model"`
	Message string `json:"prompt"`
	N       int    `json:"n"`
	Size    string `json:"size"`
	Style   string `json:"style"`
	Quality string `json:"quality"`
}

type Embed struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
	Dimensions     int         `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
}

type Response struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
	Usage map[string]interface{} `json:"usage,omitempty"`
}

type Choice struct {
	Index   int `json:"index"`
	Message *struct {
		Role             string `json:"role,omitempty"`
		Content          string `json:"content,omitempty"`
		ReasoningContent string `json:"reasoning_content,omitempty"`

		ToolCalls []Keyv[interface{}] `json:"tool_calls,omitempty"`
	} `json:"message,omitempty"`
	Delta *struct {
		Type             string `json:"type,omitempty"`
		Role             string `json:"role,omitempty"`
		Content          string `json:"content,omitempty"`
		ReasoningContent string `json:"reasoning_content,omitempty"`

		ToolCalls []Keyv[interface{}] `json:"tool_calls,omitempty"`
	} `json:"delta,omitempty"`
	FinishReason *string `json:"finish_reason"`
}
