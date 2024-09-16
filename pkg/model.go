package pkg

import (
	"encoding/json"
	"reflect"
)

type ChatCompletion struct {
	System        string              `json:"system,omitempty"`
	Messages      []Keyv[interface{}] `json:"messages"`
	Tools         []Keyv[interface{}] `json:"tools"`
	Model         string              `json:"model"`
	MaxTokens     int                 `json:"max_tokens"`
	StopSequences []string            `json:"stop"`
	Temperature   float32             `json:"temperature"`
	TopK          int                 `json:"top_k,omitempty"`
	TopP          float32             `json:"top_p,omitempty"`
	Stream        bool                `json:"stream,omitempty"`
	ToolChoice    interface{}         `json:"tool_choice"`
}

type ChatGeneration struct {
	Model   string `json:"model"`
	Message string `json:"prompt"`
	N       int    `json:"n"`
	Size    string `json:"size"`
	Style   string `json:"style"`
	Quality string `json:"quality"`
}

type EmbedRequest struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
	Dimensions     int         `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
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
	} `json:"error,omitempty"`
	Usage map[string]int `json:"usage,omitempty"`
}

type ChatChoice struct {
	Index   int `json:"index"`
	Message *struct {
		Role    string `json:"role,omitempty"`
		Content string `json:"content,omitempty"`

		ToolCalls []Keyv[interface{}] `json:"tool_calls,omitempty"`
	} `json:"message,omitempty"`
	Delta *struct {
		Role    string `json:"role,omitempty"`
		Content string `json:"content,omitempty"`

		ToolCalls []Keyv[interface{}] `json:"tool_calls,omitempty"`
	} `json:"delta,omitempty"`
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

func (kv Keyv[V]) GetSlice(key string) (values []interface{}) {
	if value, ok := kv[key]; ok {
		var v interface{} = value
		values, ok = v.([]interface{})
		if ok {
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

func (kv Keyv[V]) Is(key string, value V) (out bool) {
	if !kv.Has(key) {
		return
	}

	v, _ := kv.Get(key)
	return reflect.DeepEqual(v, value)
}

func (kv Keyv[V]) In(key string, values ...V) (out bool) {
	if !kv.Has(key) {
		return
	}

	v, _ := kv.Get(key)
	for _, value := range values {
		if reflect.DeepEqual(v, value) {
			return true
		}
	}
	return
}

func (kv Keyv[V]) String() string {
	bytes, _ := json.Marshal(kv)
	return string(bytes)
}

func (kv Keyv[V]) IsString(key string) bool {
	if value, ok := kv[key]; ok {
		var v interface{} = value
		if _, ok = v.(string); ok {
			return true
		}
	}
	return false
}

func (kv Keyv[V]) IsEmpty(key string) bool {
	return kv.GetString(key) == ""
}
