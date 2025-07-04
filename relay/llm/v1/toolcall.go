package v1

import (
	"adapter/module/fiber/context"
	"adapter/module/fiber/model"
	"adapter/module/llm"
	"adapter/module/logger"
)

func toolExecuted(ctx *context.Ctx, completion model.CompletionEntity) bool {
	logger.Sugar().Info("tool executed ...")
	exec, err := llm.ToolExecuted(ctx, completion, func(message string) (string, error) {
		completion.Stream = true
		completion.Messages = []model.CompletionMessageEntity{
			{
				"role":    "user",
				"content": message,
			},
		}

		r, err := fetch(ctx, ctx.Token, completion)
		if err != nil {
			return "", err
		}

		return waitMessage(r, llm.Cancel)
	})

	if err != nil {
		logger.Sugar().Error(err)
		Error(ctx, -1, err)
		return true
	}

	return exec
}
