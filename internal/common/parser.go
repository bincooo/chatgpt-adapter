package common

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"sort"
	"strconv"
	"strings"
)

const (
	XML_TYPE_S = iota // 普通字符串
	XML_TYPE_X        // XML标签
	XML_TYPE_I        // 注释标签
)

type XmlNode struct {
	index   int
	end     int
	tag     string
	t       int
	content string
	count   int
	attr    map[string]interface{}
	child   []XmlNode
}

type XmlParser struct {
	whiteList []string
}

// 只解析whiteList中的标签
func NewParser(whiteList []string) XmlParser {
	return XmlParser{whiteList}
}

func trimCdata(value string) string {
	// 查找从 index 开始，符合的字符串返回其下标，没有则-1
	searchStr := func(content string, index int, s string) int {
		l := len(s)
		contentL := len(content)
		for i := index + 1; i < contentL; i++ {
			if i+l > contentL {
				return -1
			}
			if content[i:i+l] == s {
				return i
			}
		}
		return -1
	}

	// 比较 index 的下一个字符串，如果相同返回 true
	nextStr := func(content string, index int, s string) bool {
		contentL := len(content)
		if index+1+len(s) >= contentL {
			return false
		}
		return content[index+1:index+1+len(s)] == s
	}

label:
	valueL := len(value)
	for i := 0; i < valueL; i++ {
		if value[i] == '<' && nextStr(value, i, "![CDATA[") {
			n := searchStr(value, i+9, "]]>")
			if n >= 0 {
				value = value[:i] + value[i+9:n] + value[n+3:]
				goto label
			}
		}
	}
	return value
}

// xml解析的简单实现
func (xml XmlParser) Parse(value string) []XmlNode {
	messageL := len(value)
	if messageL == 0 {
		return nil
	}

	var recursive func(value string) (slice []XmlNode)

	// 查找从 index 开始，符合的字节返回其下标，没有则-1
	search := func(content string, index int, ch uint8) int {
		contentL := len(content)
		for i := index + 1; i < contentL; i++ {
			if content[i] == ch {
				return i
			}
		}
		return -1
	}

	// 查找从 index 开始，符合的字符串返回其下标，没有则-1
	searchStr := func(content string, index int, s string) int {
		l := len(s)
		contentL := len(content)
		for i := index + 1; i < contentL; i++ {
			if i+l > contentL {
				return -1
			}
			if content[i:i+l] == s {
				return i
			}
		}
		return -1
	}

	// 比较 index 的下一个字节，如果相同返回 true
	next := func(content string, index int, ch uint8) bool {
		contentL := len(content)
		if index+1 >= contentL {
			return false
		}
		return content[index+1] == ch
	}

	// 比较 index 的下一个字符串，如果相同返回 true
	nextStr := func(content string, index int, s string) bool {
		contentL := len(content)
		if index+1+len(s) >= contentL {
			return false
		}
		return content[index+1:index+1+len(s)] == s
	}

	// 解析xml标签的属性
	parseAttr := func(slice []string) map[string]interface{} {
		attr := make(map[string]interface{})
		for _, it := range slice {
			n := search(it, 0, '=')
			if n <= 0 {
				if len(it) > 0 && it != "=" {
					attr[it] = true
				}
				continue
			}

			if n == len(it)-1 {
				continue
			}

			if it[n+1] == '"' && it[len(it)-1] == '"' {
				attr[it[:n]] = trimCdata(it[n+2 : len(it)-1])
			}

			s := trimCdata(it[n+1:])
			v1, err := strconv.Atoi(s)
			if err == nil {
				attr[it[:n]] = v1
				continue
			}

			v2, err := strconv.ParseFloat(s, 10)
			if err == nil {
				attr[it[:n]] = v2
				continue
			}

			v3, err := strconv.ParseBool(s)
			if err == nil {
				attr[it[:n]] = v3
				continue
			}
		}
		return attr
	}

	// 跳过 CDATA标记
	igCd := func(content string, i, j int) int {
		content = content[i:j]
		n := searchStr(content, 0, "<![CDATA[")
		if n < 0 { // 不是CD
			return j
		}

		n = searchStr(content, n, "]]>")
		if n < 0 { // 没有闭合
			return -1
		}

		if n+3 == j { // 正好是闭合的标记
			return -1
		}

		// 已经闭合
		return j
	}

	// =============
	// =============
	recursive = func(value string) (slice []XmlNode) {
		content := value
		contentL := len(content)
		var curr *XmlNode = nil
		for i := 0; i < contentL; i++ {
			// curr 的标记不完整跳过该标记，重新扫描
			if i == contentL-1 {
				if curr != nil {
					if curr.index < curr.end {
						slice = append(slice, *curr)
						i = curr.end
					} else {
						i = curr.index + len(curr.tag) + 1
					}
					curr = nil
					if i >= contentL {
						return
					}
				}
			}

			if content[i] == '<' {
				// =========================================================
				// ⬇⬇⬇⬇⬇ 结束标记 ⬇⬇⬇⬇⬇
				if curr != nil && next(content, i, '/') {
					n := search(content, i, '>')
					// 找不到 ⬇⬇⬇⬇⬇
					if n == -1 {
						// 丢弃
						curr = nil
						break
					}
					// 找不到 ⬆⬆⬆⬆⬆

					s := strings.Split(curr.tag, " ")
					if s[0] == content[i+2:n] {
						step := 2 + len(curr.tag)
						curr.t = XML_TYPE_X
						curr.end = n + 1
						// 解析xml参数
						if len(s) > 1 {
							curr.tag = s[0]
							curr.attr = parseAttr(s[1:])
						}

						str := content[curr.index+step : curr.end-len(s[0])-3]
						curr.child = recursive(str)
						curr.content = trimCdata(str)
						i = curr.end - 1

						curr.count--
						if curr.count > 0 {
							if i == contentL-1 {
								i--
							}
							continue
						}

						slice = append(slice, *curr)
						curr = nil
					}
					// ⬆⬆⬆⬆⬆ 结束标记 ⬆⬆⬆⬆⬆

					// =========================================================
					//
				} else if nextStr(content, i, "![CDATA[") {
					//
					// ⬇⬇⬇⬇⬇ <![CDATA[xxx]]> CDATA结构体 ⬇⬇⬇⬇⬇
					n := searchStr(content, i+8, "]]>")
					if n < 0 {
						i += 7
						continue
					}
					i = n + 3
					// ⬆⬆⬆⬆⬆ <![CDATA[xxx]]> CDATA结构体 ⬆⬆⬆⬆⬆

					// =========================================================
					//

				} else if nextStr(content, i, "!--") {

					//
					// ⬇⬇⬇⬇⬇ 是否是注释 <!-- xxx --> ⬇⬇⬇⬇⬇

					n := searchStr(content, i+3, "-->")
					if n < 0 {
						i += 3
						continue
					}

					node := XmlNode{index: i, end: n + 3, content: content[i : n+3], t: XML_TYPE_I}
					slice = append(slice, node)
					// ⬆⬆⬆⬆⬆ 是否是注释 <!-- xxx --> ⬆⬆⬆⬆⬆
					// 循环后置++，所以-1
					i = node.end - 1
					// =========================================================
					//

				} else {

					//
					// ⬇⬇⬇⬇⬇ 新的 XML 标记 ⬇⬇⬇⬇⬇

					idx := i
					n := search(content, idx, '>')
				label:
					if n == -1 {
						break
					}

					idx = igCd(content, i, n)
					if idx == -1 {
						idx = n
						n = search(content, idx+1, '>')
						goto label
					}

					tag := content[i+1 : n]
					// whiteList 为nil放行所有标签，否则只解析whiteList中的
					contains := xml.whiteList == nil || ContainFor(xml.whiteList, func(item string) bool {
						if strings.HasPrefix(item, "r:") {
							cmp := item[2:]
							c := regexp.MustCompile(cmp, regexp.Compiled)
							matched, err := c.MatchString(tag)
							if err != nil {
								logger.Warn("compile failed: "+cmp, err)
								return false
							}
							return matched
						}

						s := strings.Split(tag, " ")
						return item == s[0]
					})

					if !contains {
						i = n
						continue
					}

					// 这是一个自闭合的标签 <xxx />
					ch := content[n-1]
					if curr == nil && ch == '/' {
						tag = tag[:len(tag)-1]
						node := XmlNode{index: i, tag: tag, t: XML_TYPE_X}
						s := strings.Split(node.tag, " ")
						node.t = XML_TYPE_X
						node.end = n + 1
						// 解析xml参数
						if len(s) > 1 {
							node.tag = s[0]
							node.attr = parseAttr(s[1:])
						}
						slice = append(slice, node)
						i = node.end
						continue
					}

					if curr == nil {
						curr = &XmlNode{index: i, tag: tag, t: XML_TYPE_S, count: 1}
						i = n
						continue
					}

					if curr.tag == tag {
						curr.count++
						i = n
					}
					// ⬆⬆⬆⬆⬆ 新的 XML 标记 ⬆⬆⬆⬆⬆
				}
			}
		}

		return
	}

	// =========================================================
	return recursive(value)
}

func XmlFlags(ctx *gin.Context, req *pkg.ChatCompletion) []Matcher {
	matchers := NewMatchers()
	flags := pkg.Config.GetBool("flags")
	if !flags {
		return matchers
	}

	if len(req.Messages) == 0 {
		return matchers
	}

	handles := xmlFlagsToContents(ctx, req.Messages)

	abs := func(n int) int {
		if n < 0 {
			return -n
		}
		return n
	}

	for _, h := range handles {
		// 正则替换
		if h['t'] == "regex" {
			s := split(h['v'])
			if len(s) < 2 {
				continue
			}

			cmp := strings.TrimSpace(s[0])
			value := strings.TrimSpace(s[1])
			if cmp == "" {
				continue
			}

			// 忽略尾部n条
			pos, _ := strconv.Atoi(h['m'])
			if pos > -1 {
				pos = len(req.Messages) - 1 - pos
				if pos < 0 {
					pos = 0
				}
			} else {
				pos = len(req.Messages)
			}

			c := regexp.MustCompile(cmp, regexp.Compiled)
			for idx, message := range req.Messages {
				if idx < pos && message["role"] != "system" {
					replace, err := c.Replace(message.GetString("content"), value, -1, -1)
					if err != nil {
						logger.Warn("compile failed: "+cmp, err)
						continue
					}
					message["content"] = replace
				}
			}
		}

		// 深度插入
		if h['t'] == "insert" {
			i, _ := strconv.Atoi(h['i'])
			messageL := len(req.Messages)
			if h['m'] == "true" && messageL-1 < abs(i) {
				continue
			}

			pos := 0
			if i > -1 {
				// 正插
				pos = i
				if pos >= messageL {
					pos = messageL - 1
				}
			} else {
				// 反插
				pos = messageL + i
				if pos < 0 {
					pos = 0
				}
			}

			req.Messages = append(req.Messages[:pos+1], append([]pkg.Keyv[interface{}]{
				{"role": h['r'], "content": h['v']},
			}, req.Messages[pos+1:]...)...)
		}

		// matcher 流响应干预
		if h['t'] == "matcher" {
			handleMatcher(h, matchers)
		}

		// 历史记录
		if h['t'] == "histories" {
			content := strings.TrimSpace(h['v'])
			if len(content) < 2 || content[0] != '[' || content[len(content)-1] != ']' {
				continue
			}
			var baseMessages []pkg.Keyv[interface{}]
			if err := json.Unmarshal([]byte(content), &baseMessages); err != nil {
				logger.Error("histories flags handle failed: ", err)
				continue
			}

			if len(baseMessages) == 0 {
				continue
			}

			for idx := 0; idx < len(req.Messages); idx++ {
				if !strings.Contains("|system|function|", req.Messages[idx].GetString("role")) {
					req.Messages = append(req.Messages[:idx], append(baseMessages, req.Messages[idx:]...)...)
					break
				}
			}
		}
	}

	return matchers
}

// matcher 流响应干预
func handleMatcher(h map[uint8]string, matchers []Matcher) {
	find := ""
	if f, ok := h['f']; ok {
		find = f
	}
	if find == "" {
		return
	}

	findL := 5
	if l, e := strconv.Atoi(h['l']); e == nil {
		findL = l
	}

	values := split(h['v'])
	if len(values) < 2 {
		return
	}

	c := regexp.MustCompile(strings.TrimSpace(values[0]), regexp.Compiled)
	join := strings.TrimSpace(values[1])

	matchers = append(matchers, &SymbolMatcher{
		Find: find,
		H: func(index int, content string) (state int, result string) {
			r := []rune(content)
			if index+findL > len(r)-1 {
				return vars.MatMatching, content
			}
			replace, err := c.Replace(content, join, -1, -1)
			if err != nil {
				logger.Warn("compile failed: "+values[0], err)
				return vars.MatMatched, content
			}
			return vars.MatMatched, replace
		},
	})
}

func xmlFlagsToContents(ctx *gin.Context, messages []pkg.Keyv[interface{}]) (handles []map[uint8]string) {
	var (
		parser = NewParser([]string{
			"regex",
			`r:@-*\d+`,
			"debug",
			"matcher",
			"pad",      // bing中使用的标记：填充引导对话，尝试避免道歉
			"notebook", // notebook模式
			"histories",
			"tool",
		})
	)

	for _, message := range messages {
		role := message["role"]
		if role != "system" && role != "user" {
			continue
		}

		clean := func(ctx string) {
			message["content"] = strings.Replace(message.GetString("content"), ctx, "", -1)
		}

		content := message.GetString("content")
		nodes := parser.Parse(content)
		if len(nodes) == 0 {
			continue
		}

		for _, node := range nodes {
			// 注释内容删除
			if node.t == XML_TYPE_I {
				clean(content[node.index:node.end])
				continue
			}

			// 自由深度插入
			// inserts: 深度插入, i 是深度索引，v 是插入内容， o 是指令
			if node.t == XML_TYPE_X && node.tag[0] == '@' {
				c, _ := regexp.Compile(`@-*\d+`, regexp.Compiled)
				if matched, _ := c.MatchString(node.tag); matched {
					// 消息上下文次数少于插入深度时，是否忽略
					// 如不忽略，将放置在头部或者尾部
					miss := "true"
					if it, ok := node.attr["miss"]; ok {
						if v, o := it.(bool); !o || !v {
							miss = "false"
						}
					}
					// 插入元素
					r := "user"
					if it, ok := node.attr["role"]; ok {
						switch str := it.(string); str {
						case "user", "system", "assistant":
							r = str
						}
					}
					handles = append(handles, map[uint8]string{'i': node.tag[1:], 'r': r, 'v': node.content, 'm': miss, 't': "insert"})
					clean(content[node.index:node.end])
				}
				continue
			}

			// 正则替换
			// regex: v 是正则内容
			if node.t == XML_TYPE_X && node.tag == "regex" {
				order := "0" // 优先级
				if o, ok := node.attr["order"]; ok {
					order = fmt.Sprintf("%v", o)
				}

				miss := "-1"
				if m, ok := node.attr["miss"]; ok {
					miss = fmt.Sprintf("%v", m)
				}

				handles = append(handles, map[uint8]string{'m': miss, 'o': order, 'v': node.content, 't': "regex"})
				clean(content[node.index:node.end])
				continue
			}

			if node.t == XML_TYPE_X && node.tag == "matcher" {
				find := ""
				if f, ok := node.attr["find"]; ok {
					find = f.(string)
				}
				if find == "" {
					clean(content[node.index:node.end])
					continue
				}

				findLen := "5"
				if l, ok := node.attr["len"]; ok {
					findLen = fmt.Sprintf("%v", l)
				}

				handles = append(handles, map[uint8]string{'f': find, 'l': findLen, 'v': node.content, 't': "matcher"})
				clean(content[node.index:node.end])
				continue
			}

			// 开启 bing 的 pad 标记：填充引导对话，尝试避免道歉
			if node.t == XML_TYPE_X && node.tag == "pad" {
				ctx.Set("pad", true)
				clean(content[node.index:node.end])
				continue
			}

			// notebook 模式
			if node.t == XML_TYPE_X && node.tag == "notebook" {
				// 此标签默认为false， 可通过disabled属性设置开启和关闭
				disabled := false
				if e, ok := node.attr["disabled"]; ok {
					if o, k := e.(bool); k {
						disabled = o
					}
				}
				ctx.Set("notebook", !disabled)
				clean(content[node.index:node.end])
				continue
			}

			// debug 模式
			if node.t == XML_TYPE_X && node.tag == "debug" {
				ctx.Set(vars.GinDebugger, true)
				clean(content[node.index:node.end])
				continue
			}

			// 历史记录
			if node.t == XML_TYPE_X && node.tag == "histories" {
				str := strings.TrimSpace(node.content)
				if len(str) >= 2 && str[0] == '[' && str[len(str)-1] == ']' {
					handles = append(handles, map[uint8]string{'v': str, 't': "histories"})
					clean(content[node.index:node.end])
				}
				continue
			}

			if node.t == XML_TYPE_X && node.tag == "tool" {
				id := "-1"
				if e, ok := node.attr["id"]; ok {
					if o, k := e.(string); k {
						id = o
					}
				}
				tasks := false
				if e, ok := node.attr["tasks"]; ok {
					if o, k := e.(bool); k {
						tasks = o
					}
				}
				enabled := false
				if e, ok := node.attr["enabled"]; ok {
					if o, k := e.(bool); k {
						enabled = o
					}
				}

				clean(content[node.index:node.end])
				ctx.Set(vars.GinTool, pkg.Keyv[interface{}]{
					"id":      id,
					"tasks":   tasks,
					"enabled": enabled,
				})
				continue
			}
		}
	}

	if len(handles) > 0 {
		sort.Slice(handles, func(i, j int) bool {
			return handles[i]['o'] > handles[j]['o']
		})
	}
	return
}

func split(value string) []string {
	contentL := len(value)
	for i := 0; i < contentL; i++ {
		if value[i] == ':' {
			if i < 1 || value[i-1] != '\\' {
				return []string{
					strings.ReplaceAll(value[:i], "\\:", ":"), value[i+1:],
				}
			}
		}
	}
	return nil
}
