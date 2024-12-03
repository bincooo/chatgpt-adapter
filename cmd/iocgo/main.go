package main

import (
	"chatgpt-adapter/cmd/iocgo/annotation"
	"github.com/iocgo/sdk/gen"
	"github.com/iocgo/sdk/gen/tool"
)

func init() {
	// gin
	gen.Alias[annotation.GET]()
	gen.Alias[annotation.PUT]()
	gen.Alias[annotation.DEL]()
	gen.Alias[annotation.POST]()

	// cobra
	gen.Alias[annotation.Cobra]()
}

func main() {
	tool.Process()
}
