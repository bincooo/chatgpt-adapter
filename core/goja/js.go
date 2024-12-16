package goja

import (
	"chatgpt-adapter/core/gin/model"
	"encoding/json"
	"github.com/iocgo/sdk/errors"

	"chatgpt-adapter/core/logger"
	"github.com/dop251/goja"

	_ "embed"
)

var (
	//go:embed js/dist/index.js
	js string
)

func ParseMessages(messages []model.Keyv[interface{}], mode string) (newMessages []model.Keyv[interface{}], err error) {
	vm := goja.New()
	context := errors.New(func(e error) bool { err = e; return true })
	defer context.Throw()
	{
		errors.Try(context, func() error { return vm.Set("messages", messages) })
		errors.Try(context, func() error { return vm.Set("mode", mode) })
		errors.Try(context, func() error { return vm.Set("console", consoleMap()) })
		errors.Try(context, func() error { return vm.Set("JSON", jsonMap()) })
		value := errors.Try1(context, func() (goja.Value, error) { return vm.RunString(js) })
		err = vm.ExportTo(value, &newMessages)
	}
	return
}

func jsonMap() map[string]interface{} {
	return map[string]interface{}{
		"stringify": func(obj interface{}) (string, error) { value, err := json.Marshal(obj); return string(value), err },
		"parse":     func(value string) (obj interface{}, err error) { err = json.Unmarshal([]byte(value), &obj); return },
	}
}

func consoleMap() map[string]interface{} {
	return map[string]interface{}{
		"log":   func(args ...interface{}) { logger.Info(args...) },
		"debug": func(args ...interface{}) { logger.Debug(args...) },
		"error": func(args ...interface{}) { logger.Error(args...) },
	}
}
