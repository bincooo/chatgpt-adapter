package v1

import (
	"adapter/module/fiber/context"
	"errors"
	"strings"

	"adapter/module"
	"adapter/module/common"
	"adapter/module/env"
	"adapter/module/fiber/model"
)

var (
	_ module.Adapter = (*ada)(nil)

	schema = make([]model.Record[string, any], 0)
)

func init() {
	common.AddInitialized(func(env *env.Environ) {
		llm := env.Get("custom-llm")
		if slice, ok := llm.([]interface{}); ok {
			for _, it := range slice {
				var rec model.Record[string, any]
				rec, ok = it.(map[string]interface{})
				if !ok {
					continue
				}

				// validate
				_, ok = model.GetValue[string, string](rec, "reversal")
				if !ok {
					panic("`reversal` not found in config.yaml ==> custom-llm")
				}
				schema = append(schema, rec)
			}
		}
	})
}

type ada struct {
	module.BasicAdapter
}

func New() module.Adapter { return &ada{} }

func (api *ada) Models() []model.ModelEntity {
	return []model.ModelEntity{
		{
			Id:      "custom/llm?",
			Object:  "model",
			Created: 1686935002,
			By:      "custom-adapter",
		},
	}
}

// 模型匹配，当命中时执行模型函数
func (api *ada) Condition(rt module.RelayType, ctx *context.Ctx, mod string) (ret bool) {
	for _, rec := range schema {
		if prefix, ok := model.GetValue[string, string](rec, "prefix"); ok && strings.HasPrefix(mod, prefix+"/") {
			ctx.Put("custom-llm", rec)
			ctx.Put("original-model", mod[len(prefix)+1:])
			ret = true
			return
		}
	}
	return
}

// 工具选择，当返回true时不再执行模型函数
func (api *ada) ToolExecuted(ctx *context.Ctx) (ok bool, err error) {
	llm, ok := model.GetValue[string, model.Record[string, any]](ctx.Record, "custom-llm")
	if !ok {
		err = errors.New("custom-llm record not found")
		return
	}

	toolCall, ok := model.GetValue[string, bool](llm, "toolCall")
	if !ok || !toolCall {
		return
	}

	completion, ok := model.GetValue[string, *model.CompletionEntity](ctx.Record, "completion")
	if !ok {
		err = errors.New("completion record not found")
		return
	}

	ok = toolExecuted(ctx, *completion)
	return
}
