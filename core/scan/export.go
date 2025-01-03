// 该包下仅提供给iocgo工具使用的，不需要理会 Injects 的错误，在编译过程中生成

package scan

import (
	"github.com/iocgo/sdk"

	"chatgpt-adapter/cmd/cobra"
	"chatgpt-adapter/core/gin"
)

func Injects(container *sdk.Container) (err error) {
	err = cobra.Injects(container)
	if err != nil {
		return
	}

	err = gin.Injects(container)
	if err != nil {
		return
	}

	return
}
