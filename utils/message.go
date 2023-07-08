package utils

import (
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
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
	if len(slice) > 0 {
		partialResponse.Message = strings.Join(slice, "")
	}
	partialResponse.Status = vars.Closed
	return partialResponse
}
