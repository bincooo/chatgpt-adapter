package plat

import "time"

var (
	Timeout = 2 * time.Minute
	kv      = map[string]string{
		"user":     "user",
		"bot":      "assistant",
		"system":   "system",
		"function": "function",
	}
)
