package utils

import (
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"os"
	"strings"
)

func GlobalMatchers() []types.SymbolMatcher {
	slice := make([]types.SymbolMatcher, 0)
	slice = append(slice, &types.StringMatcher{
		Find: "<plot>",
		H: func(i int, content string) (int, string) {
			return types.MAT_MATCHED, strings.Replace(content, "<plot>", "", -1)
		},
	})
	slice = append(slice, &types.StringMatcher{
		Find: "</plot>",
		H: func(i int, content string) (int, string) {
			return types.MAT_MATCHED, strings.Replace(content, "</plot>", "", -1)
		},
	})
	return slice
}

func ExecMatchers(matchers []types.SymbolMatcher, raw string) string {
	for _, mat := range matchers {
		state, result := mat.Match(raw)
		if state == types.MAT_DEFAULT {
			raw = result
			continue
		}
		if state == types.MAT_MATCHING {
			raw = result
			break
		}
		if state == types.MAT_MATCHED {
			raw = result
			break
		}
	}
	return raw
}

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
