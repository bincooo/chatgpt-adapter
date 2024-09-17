package common

import (
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"strconv"
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

	globalMatchers func() []Matcher
)

// 匹配器接口
type Matcher interface {
	match(content string, over bool) (state int, result string)
}

// 字符块匹配器，只向后匹配
type SymbolMatcher struct {
	cache string // 缓存的字符
	Find  string // 字符块匹配前置，'*'则匹配任意
	// 具体的匹配实现, cache 仅在 MatMatched 状态有效
	H func(index int, content string) (state int, cache, result string)
}

func init() {
	AddInitialized(func() {
		obj := pkg.Config.Get("matcher")
		if obj == nil {
			return
		}

		if slice, ok := obj.([]interface{}); ok {
			initMatchers(slice)
		}
	})
}

func initMatchers(slice []interface{}) {
	if len(slice) == 0 {
		return
	}

	globalMatchers = func() (matchers []Matcher) {
		for _, it := range slice {
			if m, o := it.(map[string]interface{}); o {
				find, ok := m["find"]
				if !ok {
					continue
				}

				end, ok := m["end"]
				if !ok {
					end = ""
				}

				l, ok := m["len"]
				if !ok {
					l = "5"
				}

				findL, err := strconv.Atoi(l.(string))
				if err != nil {
					continue
				}

				str, ok := m["content"]
				if !ok {
					str = ""
				}

				values := split(str.(string))
				if len(values) < 2 {
					continue
				}

				c := regexp.MustCompile(strings.TrimSpace(values[0]), regexp.Compiled)
				join := strings.TrimSpace(values[1])

				var matcher *SymbolMatcher
				matcher = &SymbolMatcher{
					Find: find.(string),
					H: func(index int, content string) (state int, cache, result string) {
						if end != "" {
							if !strings.Contains(content, end.(string)) {
								return vars.MatMatching, "", content
							}
							idx := strings.LastIndex(content, end.(string))
							cache = content[idx+len(end.(string)):]
							content = content[:idx+len(end.(string))]
						} else {
							r := []rune(content)
							if index+findL > len(r)-1 {
								return vars.MatMatching, "", content
							}
						}

						logger.Infof("execute matcher[%s] content -> \n%s", matcher.Find, content)
						result, err = c.Replace(content, join, -1, -1)
						if err != nil {
							logger.Warn("compile failed: "+values[0], err)
							return vars.MatMatched, cache, content
						}
						return vars.MatMatched, cache, result
					},
				}
				matchers = append(matchers, matcher)
			}
		}
		return
	}
}

func NewMatchers() []Matcher {
	slice := make([]Matcher, 0)
	if globalMatchers != nil {
		slice = append(slice, globalMatchers()...)
	}
	return slice
}

// 中断匹配器，返回一个error channel，用于控制是否终止输出
func NewCancelMatcher(ctx *gin.Context) (chan error, []Matcher) {
	count := 0
	cancel := make(chan error, 1)
	completion := GetGinCompletion(ctx)

	var (
		user      = ""
		assistant = ""
	)

	keyv, ok := GetGinValue[pkg.Keyv[string]](ctx, vars.GinCharSequences)
	if ok {
		user = keyv.GetString("user")
		assistant = keyv.GetString("assistant")
	}

	matchers := make([]Matcher, 0)
	matchers = append(matchers, &SymbolMatcher{
		Find: "<|",
		H: func(index int, content string) (state int, _, result string) {
			if ctx.GetBool(vars.GinClose) {
				cancel <- context.Canceled
				return vars.MatMatched, "", ""
			}

			if len(content) < 13 {
				return vars.MatMatching, "", content
			}

			for _, block := range blocks {
				if strings.Contains(content, block) {
					if block == "<|assistant|>" && count == 0 {
						count++
						return vars.MatMatched, "", strings.ReplaceAll(content, "<|assistant|>", "")
					}
					cancel <- nil
					logger.Infof("matched block will closed: %s", block)
					return vars.MatMatched, "", ""
				}
			}
			return vars.MatMatched, "", content
		},
	})

	if user != "" {
		matchers = append(matchers, &SymbolMatcher{
			Find: user,
			H: func(index int, content string) (state int, _, result string) {
				if ctx.GetBool(vars.GinClose) {
					cancel <- context.Canceled
					return vars.MatMatched, "", ""
				}

				cancel <- nil
				logger.Infof("matched block will closed: %s", user)
				return vars.MatMatched, "", ""
			},
		})
	}

	if assistant != "" {
		matchers = append(matchers, &SymbolMatcher{
			Find: assistant,
			H: func(index int, content string) (state int, _, result string) {
				if ctx.GetBool(vars.GinClose) {
					cancel <- context.Canceled
					return vars.MatMatched, "", ""
				}

				cancel <- nil
				logger.Infof("matched block will closed: %s", assistant)
				return vars.MatMatched, "", ""
			},
		})
	}

	for _, value := range completion.StopSequences {
		matchers = append(matchers, &SymbolMatcher{
			Find: value,
			H: func(index int, content string) (state int, _, result string) {
				if ctx.GetBool(vars.GinClose) {
					cancel <- context.Canceled
					return vars.MatMatched, "", ""
				}

				cancel <- nil
				logger.Infof("matched block will closed: %s", value)
				return vars.MatMatched, "", ""
			},
		})
	}

	return cancel, matchers
}

// MAT_DEFAULT	没有命中，继续执行下一个
//
// MAT_MATCHING 匹配中，缓存消息不执行下一个
//
// MAT_MATCHED 	命中，不再执行下一个
func ExecMatchers(matchers []Matcher, raw string, done bool) string {
	s := vars.MatDefault
	for _, mat := range matchers {
		s, raw = mat.match(raw, done)
		if s == vars.MatDefault {
			continue
		}
		break
	}
	return raw
}

func (mat *SymbolMatcher) match(content string, over bool) (state int, result string) {
	content = mat.cache + content
	state = vars.MatDefault
	// MatDefault 没有命中
	// MatMatching 匹配中
	// MatMatched 命中了
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
			// 到这里就代表命中了，检查一下
			if strings.Contains(content, string(find)) {
				state = vars.MatMatched
			}
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
	// 没有命中，返回所有内容（包括cache）
	if state == vars.MatDefault {
		mat.cache = ""
		result = content
		return
	}

	// 还在匹配中，再次校验是否命中
	if state == vars.MatMatching {
		mat.cache = content // 缓存
		if strings.Contains(content, mat.Find) {
			state = vars.MatMatched // 命中
		} else {
			result = "" // 等待下次输入
			return
		}
	}

	if mat.H != nil {
		var leaveCache string
		state, leaveCache, result = mat.H(index, content) // 执行下游自定义处理
		if state == vars.MatMatched {                     // 处理完毕
			mat.cache = leaveCache
			return
		}
		if state == vars.MatMatching { // 还在处理中
			if over { // 已经没有后续输入了
				return vars.MatDefault, content
			}
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
