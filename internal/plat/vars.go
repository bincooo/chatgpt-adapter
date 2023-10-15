package plat

import (
	"os"
	"strings"
	"time"
)

const (
	ClackTyping = "_Typing…_"
)

var (
	Timeout = 5 * time.Minute
	kv      = map[string]string{
		"user":     "user",
		"bot":      "assistant",
		"system":   "system",
		"function": "function",
	}
)

var (
	deleteHistory = loadEnvBool("DELETE_HISTORY", false)
)

func loadEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	return strings.TrimSpace(strings.ToLower(value)) == "true"
}
