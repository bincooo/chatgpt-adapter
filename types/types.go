package types

type PartialResponse struct {
	Status  string
	Message string
	Error   error
}

// 会话上下文
type ConversationContext struct {
	Id        string
	Bot       string
	Prompt    string
	Token     string
	Preset    string
	Format    string
	AppId     string
	MaxTokens int

	BaseURL string
	Model   string
	Chain   string

	Data any
}

type Limiter interface {
	Join(ConversationContext, chan PartialResponse) error
	Remove(bot string)
	RegChain(name string, inter Interceptor) error
}

// 机器人管理器
type BotManager interface {
	// 回复
	Reply(ctx ConversationContext, response chan PartialResponse) PartialResponse
	// 添加机器人
	Add(key string, bot Bot)
	// 删除机器人
	Remove(key string)

	RegChain(name string, inter Interceptor) error
}

// 机器人
type Bot interface {
	Reply(ctx ConversationContext) chan PartialResponse
	Reset(id string) bool
}

// 拦截处理器
type Interceptor interface {
	// true 继续下一个
	Before(bot *Bot, ctx *ConversationContext) bool
	// true 继续下一个
	After(bot *Bot, ctx *ConversationContext, response string) bool
}

// 空实现
type BaseInterceptor struct {
}

func (*BaseInterceptor) Before(*Bot, *ConversationContext) bool {
	return true
}

func (*BaseInterceptor) After(*Bot, *ConversationContext, string) bool {
	return true
}
