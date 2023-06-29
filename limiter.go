package MiaoX

import (
	"errors"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ConversationStack struct {
	conversation types.ConversationContext
	response     chan types.PartialResponse
}

// 通用限流器
// 内置分组限流和全局限流
type CommonLimiter struct {
	lmt  *Limiter
	gLmt *GroupLimiter
}

// 全局限流器
type Limiter struct {
	sync.RWMutex

	max   int                 // 最大队列数量
	stack []ConversationStack // 会话栈存放列表
	mgr   types.BotManager    // AI管理器
}

// 群组限流器
type GroupLimiter struct {
	kv    map[string]*Limiter
	botKv map[string][]*Limiter
}

func NewCommonLimiter() *CommonLimiter {
	return &CommonLimiter{
		lmt:  NewLimiter(),
		gLmt: NewGroupLimiter(),
	}
}

func NewLimiter() *Limiter {
	lmt := Limiter{
		max:   3,
		stack: []ConversationStack{},
		mgr:   NewBotManager(),
	}
	go lmt.Run()
	return &lmt
}

func NewGroupLimiter() *GroupLimiter {
	lmt := GroupLimiter{
		kv:    make(map[string]*Limiter, 0),
		botKv: make(map[string][]*Limiter, 0),
	}
	return &lmt
}

func (cLmt *CommonLimiter) Join(context types.ConversationContext, response chan types.PartialResponse) error {
	lmt := cLmt.matchLimiter(context.Bot)
	if lmt == nil {
		return errors.New("未知的`AI`类型")
	}
	return lmt.Join(context, response)
}

func (cLmt *CommonLimiter) Remove(bot string) {
	lmt := cLmt.matchLimiter(bot)
	if lmt != nil {
		lmt.Remove(bot)
	}
}

func (cLmt *CommonLimiter) matchLimiter(bot string) types.Limiter {
	switch bot {
	case vars.OpenAIAPI, vars.Claude, vars.Bing:
		return cLmt.gLmt
	case vars.OpenAIWeb:
		return cLmt.lmt
	default:
		return nil
	}
}

// ==== Limiter =====

func (lmt *Limiter) Join(context types.ConversationContext, response chan types.PartialResponse) error {
	lmt.Lock()
	defer lmt.Unlock()

	if len(lmt.stack) > lmt.max {
		return errors.New("忙不过来了，CPU都得冒烟~")
	}

	lmt.stack = append(lmt.stack, ConversationStack{
		conversation: context,
		response:     response,
	})
	logrus.Info("[MiaoX] - 已加入队列")
	return nil
}

func (lmt *Limiter) Remove(bot string) {
	lmt.mgr.Remove(bot)
}

// ==== End =====

// ==== GroupLimiter =====

func (gLmt *GroupLimiter) Join(context types.ConversationContext, response chan types.PartialResponse) error {
	// 群和好友各自用一个限流
	value, ok := gLmt.kv[context.Id]
	if !ok {
		value = NewLimiter()
		gLmt.kv[context.Id] = value
	}

	botV, ok := gLmt.botKv[context.Bot]
	if !ok {
		gLmt.botKv[context.Bot] = []*Limiter{value}
	} else {
		gLmt.botKv[context.Bot] = append(botV, value)
	}

	return value.Join(context, response)
}

func (gLmt *GroupLimiter) Remove(bot string) {
	values, ok := gLmt.botKv[bot]
	if ok {
		for _, value := range values {
			value.Remove(bot)
		}
	}
}

// ==== End ====

func (lmt *Limiter) Run() {
	waitTimeout := time.Second
	for {
		if len(lmt.stack) == 0 {
			time.Sleep(waitTimeout)
			continue
		}

		lmt.Lock()
		s := lmt.stack[0]
		lmt.stack = lmt.stack[1:len(lmt.stack)]
		lmt.Unlock()

		logrus.Info("[MiaoX] - 正在应答，ID = ", s.conversation.Id, ", Bot = ", s.conversation.Bot)
		lmt.mgr.Reply(s.conversation, s.response)
		time.Sleep(waitTimeout)
	}
}
