package wasm

import (
	"github.com/wasmerio/wasmer-go/wasmer"
	"os"
)

type Instance *wasmer.Instance
type NativeFunction wasmer.NativeFunction

func New(path string) (instance Instance, err error) {
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return
	}
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		return
	}

	instance, err = wasmer.NewInstance(module, wasmer.NewImportObject())
	return
}
