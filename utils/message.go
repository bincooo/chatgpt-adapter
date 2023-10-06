package utils

import (
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"os"
	"strings"
)

func MergeFullMessage(message chan types.PartialResponse) types.PartialResponse {
	var partialResponse types.PartialResponse
	var slice []string
	for {
		if response, ok := <-message; ok {
			if response.Error != nil {
				partialResponse = response
				break
			}
			slice = append(slice, response.Message)
		} else {
			break
		}
	}
	if partialResponse.Error != nil {
		return partialResponse
	}
	if len(slice) > 0 {
		partialResponse.Message = strings.Join(slice, "")
	}
	partialResponse.Status = vars.Closed
	return partialResponse
}

func LoadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}
