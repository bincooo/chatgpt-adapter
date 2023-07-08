package types

type PartialResponse struct {
	Status  string
	Message string
	Error   error
}

// 会话上下文
type ConversationContext struct {
	Id        string // 唯一Id
	Bot       string // AI机器人类型
	Prompt    string // 文本
	Token     string // 凭证
	Preset    string // 预设模版
	Format    string // 消息模板
	AppId     string // 设备Id（claude/bing？）
	MaxTokens int    // 最大对话Tokens长度

	BaseURL string // 转发URL
	Model   string // AI模型类型
	Chain   string // 拦截处理器链
	Proxy   string // 本地代理

	Data any // 拓展数据
}

type Limiter interface {
	Join(ctx ConversationContext, handle func(response chan PartialResponse)) error
	Remove(id string, bot string)
	RegChain(name string, inter Interceptor) error
}

// 机器人管理器
type BotManager interface {
	// 回复
	Reply(ctx ConversationContext, handle func(response chan PartialResponse)) PartialResponse
	// 添加机器人
	Add(key string, bot Bot)
	// 删除机器人
	Remove(id string, key string)

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
