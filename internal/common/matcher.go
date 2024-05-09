package common

import (
	"strings"
)

const (
	MAT_DEFAULT  int = iota // 接收字符，并执行下一个匹配器
	MAT_MATCHING            // 匹配中
	MAT_MATCHED             // 匹配器命中
)

// 匹配器，匹配常量符号流式结果处理
type Matcher interface {
	match(content string) (state int, result string)
}

// 符号串符号适配器
type SymbolMatcher struct {
	cache string
	Find  string
	H     func(index int, content string) (state int, result string)
}

func NewMatchers() []Matcher {
	slice := make([]Matcher, 0)
	// todo 内置一些过滤器
	return slice
}

func ExecMatchers(matchers []Matcher, raw string) string {
	for _, mat := range matchers {
		state, result := mat.match(raw)
		if state == MAT_DEFAULT {
			raw = result
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
