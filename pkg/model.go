package pkg

type ChatCompletion struct {
	Messages      []Keyv[interface{}] `json:"messages"`
	Tools         []Keyv[interface{}] `json:"tools"`
	Model         string              `json:"model"`
	MaxTokens     int                 `json:"max_tokens"`
	StopSequences []string            `json:"stop_sequences"`
	Temperature   float32             `json:"temperature"`
	TopK          int                 `json:"topK"`
	TopP          float32             `json:"topP"`
	Stream        bool                `json:"stream"`
	ToolChoice    string              `json:"tool_choice"`
}

type ChatGeneration struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
	Style  string `json:"style"`
}

type Keyv[V any] map[string]V

type ChatResponse struct {
	Id      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Usage map[string]int `json:"usage"`
}

type ChatChoice struct {
	Index   int `json:"index"`
	Message *struct {
		Role    string `json:"role"`
		Content string `json:"content"`

		ToolCalls []Keyv[interface{}] `json:"tool_calls"`
	} `json:"message"`
	Delta *struct {
		Role    string `json:"role"`
		Content string `json:"content"`

		ToolCalls []Keyv[interface{}] `json:"tool_calls"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

func (kv Keyv[V]) Set(key string, value V) {
	kv[key] = value
}

func (kv Keyv[V]) Get(key string) (V, bool) {
	value, ok := kv[key]
	return value, ok
}

func (kv Keyv[V]) Has(key string) bool {
	_, ok := kv.Get(key)
	return ok
}

func (kv Keyv[V]) GetKeyv(key string) (out Keyv[interface{}]) {
	if value, ok := kv[key]; ok {
		var v interface{} = value
		if val, o := v.(map[string]interface{}); o {
			out = val
			return
		}
	}
	return nil
}

func (kv Keyv[V]) GetString(key string) (out string) {
	if value, ok := kv[key]; ok {
		var v interface{} = value
		if out, ok = v.(string); ok {
			return
		}
	}
	return
}
