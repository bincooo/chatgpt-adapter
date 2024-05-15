package common

import (
	"strings"
)

const (
	MAT_DEFAULT  int = iota // 执行下一个匹配器
	MAT_MATCHING            // 匹配中, 字符被缓存
	MAT_MATCHED             // 匹配器命中，不再执行下一个
)

var (
	blocks = []string{
		"<|system|>",
		"<|user|>",
		"<|assistant|>",
		"<|function|>",
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
func NewCancelMather() (chan error, Matcher) {
	count := 0
	cancel := make(chan error, 1)
	return cancel, &SymbolMatcher{
		Find: "<|",
		H: func(index int, content string) (state int, result string) {
			if len(content) < 13 {
				return MAT_MATCHING, content
			}

			for _, block := range blocks {
				if strings.Contains(content, block) {
					if block == "<|assistant|>" && count == 0 {
						count++
						return MAT_MATCHED, strings.ReplaceAll(content, "<|assistant|>", "")
					}
					cancel <- nil
					return MAT_MATCHED, ""
				}
			}
			return MAT_MATCHED, content
		},
	}
}

func ExecMatchers(matchers []Matcher, raw string) string {
	// MAT_DEFAULT	没有命中，继续执行下一个
	// MAT_MATCHING 匹配中，缓存消息不执行下一个
	// MAT_MATCHED 	命中，不再执行下一个
	for _, mat := range matchers {
		state, result := mat.match(raw)
		if state == MAT_DEFAULT {
			continue
		}
		if state == MAT_MATCHING {
			raw = result
			break
		}
		if state == MAT_MATCHED {
			raw = result
			break
		}
	}
	return raw
}

func (mat *SymbolMatcher) match(content string) (state int, result string) {
	content = mat.cache + content
	state = MAT_DEFAULT

	var (
		index = 0
		find  = []rune(mat.Find)
		rc    = []rune(content)

		pos = 0
		idx = -1
	)

	if mat.Find == "" || mat.Find == "*" {
		state = MAT_MATCHED
		goto state
	}

	for index = range rc {
		var ch rune
		if len(find) <= pos {
			state = MAT_MATCHED
			if mat.H != nil {
				break
			}
			continue
		}
		ch = find[pos]
		if ch != rc[index] {
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

state:
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
				mat.cache = result
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
