package types

type RequestDTO struct {
	//Prompt        string              `json:"prompt"`
	Messages      []map[string]string `json:"messages"`
	Model         string              `json:"model"`
	MaxTokens     int                 `json:"max_tokens_to_sample"`
	StopSequences []string            `json:"stop_sequences"`
	Temperature   float32             `json:"temperature"`
	Stream        bool                `json:"stream"`
}
