package response

import (
	"chatgpt-adapter/core/logger"
	encoder "github.com/samber/go-gpt-3-encoder"
)

func CalcTokens(content string) int {
	resolver, err := encoder.NewEncoder()
	if err != nil {
		logger.Error(err)
		return 0
	}
	result, err := resolver.Encode(content)
	if err != nil {
		logger.Error(err)
		return 0
	}
	return len(result)
}

func CalcUsageTokens(content string, previousTokens int) map[string]interface{} {
	tokens := CalcTokens(content)
	return map[string]interface{}{
		"completion_tokens": tokens,
		"prompt_tokens":     previousTokens,
		"total_tokens":      previousTokens + tokens,
	}
}
