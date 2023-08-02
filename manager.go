package MiaoX

import (
	"errors"
	"github.com/bincooo/MiaoX/internal/chain"
	"github.com/bincooo/MiaoX/internal/plat"
	"github.com/bincooo/MiaoX/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"github.com/sirupsen/logrus"
	"strings"
)

type CommonBotManager struct {
	bots  map[string]types.Bot
	chain *chain.Chain
}

func NewBotManager() types.BotManager {
	return &CommonBotManager{
		chain: chain.New(),
		bots:  map[string]types.Bot{},
	}
}

// 管理器应答消息
func (mgr *CommonBotManager) Reply(ctx types.ConversationContext, handle func(types.PartialResponse)) types.PartialResponse {

	h := func(partialResponse types.PartialResponse) types.PartialResponse {
		if handle != nil {
			handle(partialResponse)
		}
		return partialResponse
	}

	if ctx.Prompt == "" {
		return h(types.PartialResponse{Error: errors.New("请输入你的文本内容"), Status: vars.Closed})
	}

	if _, ok := mgr.bots[ctx.Bot]; !ok {
		if err := mgr.makeBot(ctx.Bot); err != nil {
			return h(types.PartialResponse{Error: err, Status: vars.Closed})
		}
	}

	if _, ok := mgr.bots[ctx.Bot]; !ok {
		return h(types.PartialResponse{Error: errors.New("未知的AI类型: " + ctx.Bot), Status: vars.Closed})
	}

	bot := mgr.bots[ctx.Bot]
	if strings.Contains("|重置|重置会话|重置对话|reset|", "|"+ctx.Prompt+"|") {
		var result string
		if bot.Remove(ctx.Id) {
			result = "已重置，开始新的对话吧"
		} else {
			result = "重置失败"
		}

		store.DeleteMessages(ctx.Id)
		return h(types.PartialResponse{Message: result, Status: vars.Closed})
	}
	return mgr.replyConversation(bot, handle, ctx)
}

// 添加机器人
func (mgr *CommonBotManager) Add(name string, bot types.Bot) {
	mgr.bots[name] = bot
}

// 删除机器人
func (mgr *CommonBotManager) Remove(uid string, name string) {
	store.DeleteMessages(uid)
	if bot, ok := mgr.bots[name]; ok {
		bot.Remove(uid)
	}
	//delete(mgr.bots, name)
}

// 构建机器人
func (mgr *CommonBotManager) makeBot(bot string) error {
	switch bot {
	case vars.OpenAIAPI:
		mgr.Add(bot, plat.NewOpenAIAPIBot())
	case vars.OpenAIWeb:
		mgr.Add(bot, plat.NewOpenAIWebBot())
	case vars.Claude:
		mgr.Add(bot, plat.NewClaudeBot())
	case vars.Bing:
		mgr.Add(bot, plat.NewBingBot())
	default:
		logrus.Error("未定义的AI类型：" + bot)
	}
	return nil
}

func (mgr *CommonBotManager) replyConversation(bot types.Bot, handle func(types.PartialResponse), ctx types.ConversationContext) types.PartialResponse {
	h := func(partialResponse types.PartialResponse) types.PartialResponse {
		if handle != nil {
			handle(partialResponse)
		}
		return partialResponse
	}

	h(types.PartialResponse{Status: vars.Begin})
	mgr.chain.Before(bot, &ctx)

	var err error
	var slice []types.PartialResponse
	chanResponse := bot.Reply(ctx)
	for {
		if partialResponse, ok := <-chanResponse; ok {
			h(partialResponse)
			if partialResponse.Error != nil {
				err = partialResponse.Error
				break
			}
			slice = append(slice, partialResponse)
		} else {
			break
		}
	}

	var message string
	for _, value := range slice {
		message += value.Message
	}
	mgr.chain.After(bot, &ctx, message)
	partialResponse := types.PartialResponse{Message: message, Error: err, Status: vars.Closed}
	//if err != nil {
	//	h(partialResponse)
	//}
	return partialResponse
}

func (mgr *CommonBotManager) RegChain(name string, inter types.Interceptor) error {
	if mgr.chain.Has(name) {
		return errors.New("拦截处理器`" + name + "`已存在")
	}

	mgr.chain.Set(name, inter)
	return nil
}
