package chain

import (
	"github.com/bincooo/MiaoX/internal/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/utils"
	"github.com/jinzhu/copier"
	"github.com/sirupsen/logrus"
	"time"
)

// 预加载预设（协助openai-web和claude这类需要预先发送预设的AI）
type AssistInterceptor struct {
	types.BaseInterceptor
}

func (c *AssistInterceptor) Before(bot types.Bot, ctx *types.ConversationContext) bool {
	messages := store.GetMessages(ctx.Id)
	if len(messages) == 0 && ctx.Preset != "" {
		// 发送预设模版
		var context types.ConversationContext
		if err := copier.Copy(&context, ctx); err != nil {
			logrus.Error(err)
			return true
		}

		context.Prompt = ctx.Preset
		message := bot.Reply(context)
		partialResponse := utils.MergeFullMessage(message)
		if partialResponse.Error != nil {
			logrus.Error(partialResponse.Error)
		}

		logrus.Info("\n*** FIRST PRESET RESULT ***\n", partialResponse.Message, "\n*******************")
		time.Sleep(time.Second)
		logrus.Info("[MiaoX] - 加载预设完毕")
	}
	return true
}
