package common

import (
	encoder "github.com/samber/go-gpt-3-encoder"
	"github.com/sirupsen/logrus"
)

// 计算content的token长度
func CalcTokens(content string) int {
	resolver, err := encoder.NewEncoder()
	if err != nil {
		logrus.Error(err)
		return 0
	}
	result, err := resolver.Encode(content)
	if err != nil {
		logrus.Error(err)
		return 0
	}
	return len(result)
}

func CalcUsageTokens(content string, previousTokens int) map[string]int {
	tokens := CalcTokens(content)
	return map[string]int{
		"completion_tokens": tokens,
		"prompt_tokens":     previousTokens,
		"total_tokens":      previousTokens + tokens,
	}
}
