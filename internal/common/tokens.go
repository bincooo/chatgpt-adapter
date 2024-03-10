package common

import (
	encoder "github.com/samber/go-gpt-3-encoder"
	"github.com/sirupsen/logrus"
)

// 计算prompt的token长度
func CalcTokens(prompt string) int {
	resolver, err := encoder.NewEncoder()
	if err != nil {
		logrus.Error(err)
		return 0
	}
	result, err := resolver.Encode(prompt)
	if err != nil {
		logrus.Error(err)
		return 0
	}
	return len(result)
}
