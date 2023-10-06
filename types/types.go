package types

import "strings"

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

	H func(partialResponse any) func(*CacheBuffer) error // 自定义流处理器
}

type CustomCacheHandler = func(partialResponse any) func(*CacheBuffer) error

type Limiter interface {
	Join(ctx ConversationContext, handle func(response PartialResponse)) error
	Remove(id string, bot string)
	RegChain(name string, inter Interceptor) error
}

// 机器人管理器
type BotManager interface {
	// 回复
	Reply(ctx ConversationContext, handle func(response PartialResponse)) PartialResponse
	// 添加机器人
	Add(key string, bot Bot)
	// 删除机器人
	Remove(id string, key string)

	RegChain(name string, inter Interceptor) error
}

// 机器人
type Bot interface {
	Reply(ctx ConversationContext) chan PartialResponse
	Remove(id string) bool
}

// 拦截处理器，预处理用户输入以及bot输出
type Interceptor interface {
	// true 继续下一个
	Before(bot Bot, ctx *ConversationContext) (bool, error)
	// true 继续下一个
	After(bot Bot, ctx *ConversationContext, response string) (bool, error)
}

// 空实现
type BaseInterceptor struct {
}

func (*BaseInterceptor) Before(Bot, *ConversationContext) (bool, error) {
	return true, nil
}

func (*BaseInterceptor) After(Bot, *ConversationContext, string) (bool, error) {
	return true, nil
}

// =============

const (
	MAT_DEFAULT int = iota
	MAT_MATCHING
	MAT_MATCHED
)

// 符号匹配器，匹配常量符号并流式处理
type SymbolMatcher interface {
	Match(content string) (state int, result string)
}

// 字符串符号适配器
type StringMatcher struct {
	cache string
	Find  string
	H     func(index int, content string) (state int, result string)
}

func (mat *StringMatcher) Match(content string) (state int, result string) {
	if mat.Find == "" {
		panic("`StringMatcher::Find` is empty")
	}

	content = mat.cache + content

	f := []rune(mat.Find)
	r := []rune(content)

	pos := 0
	idx := -1

	index := 0
	state = MAT_DEFAULT

	for index = range r {
		var fCh rune
		if len(f) <= pos {
			state = MAT_MATCHED
			if mat.H != nil {
				break
			}
			continue
		}
		fCh = f[pos]
		if fCh != r[index] {
			pos = 0
			idx = -1
			state = MAT_DEFAULT
			continue
		}
		if idx == -1 || idx == index-1 {
			pos++
			idx = index
			state = MAT_MATCHING
			continue
		}
	}

	if state == MAT_DEFAULT {
		mat.cache = ""
		result = content
		return
	}

	if state == MAT_MATCHING {
		mat.cache = content
		if strings.HasSuffix(content, mat.Find) {
			state = MAT_MATCHED
		} else {
			result = ""
			return
		}
	}

	if state == MAT_MATCHED {
		if mat.H != nil {
			state, result = mat.H(index, content)
			if state == MAT_MATCHED {
				mat.cache = ""
				return
			}
			if state == MAT_MATCHING {
				mat.cache = content
				return state, ""
			}
			return state, content
		} else {
			result = content
			mat.cache = ""
		}
	}

	return
}
