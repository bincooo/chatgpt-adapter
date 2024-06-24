package common

import (
	"context"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/gin-gonic/gin"
	"strings"
)

var (
	blocks = []string{
		"<|system|>",
		"<|user|>",
		"<|assistant|>",
		"<|function|>",
		"<|tool|>",
		"<|end|>",
	}
)

// 匹配器接口
type Matcher interface {
	match(content string) (state int, result string)
}

// 字符块匹配器，只向后匹配
type SymbolMatcher struct {
	cache string // 缓存的字符
	Find  string // 字符块匹配前置，'*'则匹配任意
	// 具体的匹配实现
	H func(index int, content string) (state int, result string)
}

func NewMatchers() []Matcher {
	slice := make([]Matcher, 0)
	// todo 内置一些过滤器
	return slice
}

// 中断匹配器，返回一个error channel，用于控制是否终止输出
func NewCancelMather(ctx *gin.Context) (chan error, Matcher) {
	count := 0
	cancel := make(chan error, 1)

	newBlocks := make([]string, 0)
	newBlocks = append(newBlocks, blocks...)

	keyv, ok := GetGinValue[pkg.Keyv[string]](ctx, vars.GinCharSequences)
	if ok {
		user := keyv.GetString("user")
		assistant := keyv.GetString("assistant")
		if user != "" {
			newBlocks = append(newBlocks, user)
		}
		if assistant != "" {
			newBlocks = append(newBlocks, assistant)
		}
	}

	return cancel, &SymbolMatcher{
		Find: "<|",
		H: func(index int, content string) (state int, result string) {
			if ctx.GetBool(vars.GinClose) {
				cancel <- context.Canceled
				return vars.MatMatched, ""
			}

			if len(content) < 13 {
				return vars.MatMatching, content
			}

			for _, block := range newBlocks {
				if strings.Contains(content, block) {
					if block == "<|assistant|>" && count == 0 {
						count++
						return vars.MatMatched, strings.ReplaceAll(content, "<|assistant|>", "")
					}
					cancel <- nil
					return vars.MatMatched, ""
				}
			}
			return vars.MatMatched, content
		},
	}
}

func ExecMatchers(matchers []Matcher, raw string) string {
	// MAT_DEFAULT	没有命中，继续执行下一个
	// MAT_MATCHING 匹配中，缓存消息不执行下一个
	// MAT_MATCHED 	命中，不再执行下一个
	for _, mat := range matchers {
		state, result := mat.match(raw)
		if state == vars.MatDefault {
			raw = result
			continue
		}
		if state == vars.MatMatching {
			raw = result
			break
		}
		if state == vars.MatMatched {
			raw = result
			break
		}
	}
	return raw
}

func (mat *SymbolMatcher) match(content string) (state int, result string) {
	content = mat.cache + content
	state = vars.MatDefault

	var (
		index = 0
		find  = []rune(mat.Find)
		rc    = []rune(content)

		pos = 0
		idx = -1
	)

	if mat.Find == "" || mat.Find == "*" {
		state = vars.MatMatched
		goto state
	}

	for index = range rc {
		var ch rune
		if len(find) <= pos {
			state = vars.MatMatched
			if mat.H != nil {
				break
			}
			continue
		}
		ch = find[pos]
		if ch != rc[index] {
			pos = 0
			idx = -1
			state = vars.MatDefault
			continue
		}
		if idx == -1 || idx == index-1 {
			pos++
			idx = index
			state = vars.MatMatching
			continue
		}
	}

state:
	if state == vars.MatDefault {
		mat.cache = ""
		result = content
		return
	}

	if state == vars.MatMatching {
		mat.cache = content
		if strings.HasSuffix(content, mat.Find) {
			state = vars.MatMatched
		} else {
			result = ""
			return
		}
	}

	if mat.H != nil {
		state, result = mat.H(index, content)
		if state == vars.MatMatched {
			mat.cache = ""
			return
		}
		if state == vars.MatMatching {
			mat.cache = result
			return state, ""
		}
		return state, content
	} else {
		result = content
		mat.cache = ""
	}

	return
}
