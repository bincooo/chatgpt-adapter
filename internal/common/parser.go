package common

import (
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/bincooo/coze-api"
	regexp "github.com/dlclark/regexp2"
	"github.com/dop251/goja"
	"github.com/gin-gonic/gin"
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
					attr[strings.TrimSpace(it)] = true
				}
				continue
			}

			if n == len(it)-1 {
				continue
			}

			if it[n+1] == '"' && it[len(it)-1] == '"' {
				attr[strings.TrimSpace(it[:n])] = trimCdata(it[n+2 : len(it)-1])
			}

			s := trimCdata(it[n+1:])
			v1, err := strconv.Atoi(s)
			if err == nil {
				attr[strings.TrimSpace(it[:n])] = v1
				continue
			}

			v2, err := strconv.ParseFloat(s, 10)
			if err == nil {
				attr[strings.TrimSpace(it[:n])] = v2
				continue
			}

			v3, err := strconv.ParseBool(s)
			if err == nil {
				attr[strings.TrimSpace(it[:n])] = v3
				continue
			}
		}
		return attr
	}

	// 跳过 CDATA标记
	igCd := func(content string, i, j int) int {
		ctx := content[i:j]
		n := searchStr(ctx, 0, "<![CDATA[")
		if n < 0 { // 不是CD
			return j
		}

		n = searchStr(ctx, n, "]]>")
		if n < 0 { // 没有闭合
			return -1
		}

		if n+3 == j { // 正好是闭合的标记
			return -1
		}

		// 已经闭合
		return i + n + 3
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

					s := split2(curr.tag, " ")
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

					tmp := igCd(content, idx, n)
					if tmp == -1 {
						tmp = n
						n = search(content, tmp+1, '>')
						goto label
					}

					if tmp < n {
						idx = tmp
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

func XmlFlags(ctx *gin.Context, completion *pkg.ChatCompletion, cb func(str string)) ([]Matcher, error) {
	matchers := NewMatchers(cb)
	flags := pkg.Config.GetBool("flags")
	if !flags {
		if err := handleClaudeMessages(ctx, *completion); err != nil {
			return nil, err
		}
		return matchers, nil
	}

	if len(completion.Messages) == 0 {
		return matchers, nil
	}

	token := ctx.GetString("token")
	handles := xmlFlagsToContents(ctx, completion.Messages, IsClaude(ctx, token, completion.Model))

	for _, h := range handles {
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

			for idx := 0; idx < len(completion.Messages); idx++ {
				if completion.Messages[idx].In("role", "assistant", "user") {
					completion.Messages = append(completion.Messages[:idx], append(baseMessages, completion.Messages[idx:]...)...)
					break
				}
			}
		}
	}

	if err := handleClaudeMessages(ctx, *completion); err != nil {
		return nil, err
	}
	values, ok := GetGinValue[[]pkg.Keyv[interface{}]](ctx, vars.GinClaudeMessages)
	if ok {
		_ = xmlFlagsToContents(ctx, values, true)
	}

	return matchers, nil
}

func IsClaude(ctx *gin.Context, token, model string) bool {
	key := "__is-claude__"
	if ctx.GetBool(key) {
		return true
	}

	if model == "coze/websdk" {
		model = pkg.Config.GetString("coze.websdk.model")
		return model == coze.ModelClaude35Sonnet_200k || model == coze.ModelClaude3Haiku_200k
	}

	isc := strings.Contains(strings.ToLower(model), "claude")
	if isc {
		ctx.Set(key, true)
		return true
	}

	if strings.HasPrefix(model, "coze/") {
		values := strings.Split(model[5:], "-")
		if len(values) > 3 && "w" == values[3] && strings.Contains(token, "[claude=true]") {
			ctx.Set(key, true)
			return true
		}

		return false
	}

	return isc
}

func handleClaudeMessages(ctx *gin.Context, completion pkg.ChatCompletion) (err error) {
	// claude messages
	token := ctx.GetString("token")
	if IsClaude(ctx, token, completion.Model) {
		vm := goja.New()
		_ = vm.Set("messages", completion.Messages)
		_ = vm.Set("console", map[string]interface{}{
			"log": func(args ...interface{}) {
				fmt.Println(args...)
			},
		})
		_ = vm.Set("JSON", map[string]interface{}{
			"stringify": func(obj interface{}) (string, error) {
				value, e := json.Marshal(obj)
				return string(value), e
			},
			"parse": func(value string) (obj interface{}, e error) {
				e = json.Unmarshal([]byte(value), &obj)
				return
			},
		})

		v, e := vm.RunString(vars.Script)
		if e != nil {
			err = e
			return
		}

		var msgs []pkg.Keyv[interface{}]
		err = vm.ExportTo(v, &msgs)
		if err != nil {
			return
		}

		ctx.Set(vars.GinClaudeMessages, msgs)
	}

	return nil
}

// matcher 流响应干预
func handleMatcher(h map[uint8]string, matchers []Matcher) []Matcher {
	find := ""
	if f, ok := h['f']; ok {
		find = f
	}
	if find == "" {
		return matchers
	}

	end := ""
	if e, ok := h['e']; ok {
		end = e
	}

	findL := 5
	if l, e := strconv.Atoi(h['l']); e == nil {
		findL = l
	}

	values := split(h['v'])
	if len(values) < 2 {
		return matchers
	}

	c := regexp.MustCompile(strings.TrimSpace(values[0]), regexp.Compiled)
	join := strings.TrimSpace(values[1])

	matchers = append(matchers, &SymbolMatcher{
		Find: find,
		H: func(index int, content string) (state int, _, result string) {
			var err error
			if end != "" {
				if !strings.Contains(content, end) {
					return vars.MatMatching, "", content
				}
			} else {
				r := []rune(content)
				if index+findL > len(r)-1 {
					return vars.MatMatching, "", content
				}
			}

			result, err = c.Replace(content, join, -1, -1)
			if err != nil {
				logger.Warn("compile failed: "+values[0], err)
				return vars.MatMatched, "", content
			}

			return vars.MatMatched, "", result
		},
	})

	return matchers
}

func xmlFlagsToContents(ctx *gin.Context, messages []pkg.Keyv[interface{}], isc bool) (handles []map[uint8]string) {
	var (
		parser = NewParser([]string{
			"debug",
			"pad",      // bing中使用的标记：填充引导对话，尝试避免道歉
			"notebook", // notebook模式
			"histories",
			"char_sequences", // 角色序列映射
			"tool",
			"echo", // 不与AI交互，仅获取处理后的上下文
		})
	)

	for _, message := range messages {
		if !message.In("role", "system", "user") {
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
				//clean(content[node.index:node.end])
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

			// 不与AI交互，仅获取处理后的上下文
			if node.t == XML_TYPE_X && node.tag == "echo" {
				ctx.Set(vars.GinEcho, true)
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

			// 角色序列映射
			if node.t == XML_TYPE_X && node.tag == "char_sequences" {
				var (
					user      = ""
					assistant = ""
				)
				if e, ok := node.attr["user"]; ok {
					if o, k := e.(string); k {
						user = o
					}
				}
				if e, ok := node.attr["assistant"]; ok {
					if o, k := e.(string); k {
						assistant = o
					}
				}
				ctx.Set(vars.GinCharSequences, pkg.Keyv[string]{
					"user":      user,
					"assistant": assistant,
				})
				clean(content[node.index:node.end])
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

	//if len(handles) > 0 {
	//	sort.Slice(handles, func(i, j int) bool {
	//		return handles[i]['o'] > handles[j]['o']
	//	})
	//}
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

func split2(content string, delimiter string) (result []string) {
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
	//nextStr := func(content string, index int, s string) bool {
	//	contentL := len(content)
	//	if index+1+len(s) >= contentL {
	//		return false
	//	}
	//	return content[index+1:index+1+len(s)] == s
	//}

	contentL := len(content)
	pos := 0
	for i := 0; i < contentL; i++ {
		n := searchStr(content, i, delimiter)
		if n == -1 {
			break
		}

		if n >= 0 {
			tmp := searchStr(content, pos, "![CDATA[")
			if tmp >= 0 && n > tmp {
				tmp = searchStr(content, tmp+9, "]]>")
				if tmp >= 0 && n < tmp+3 {
					i = tmp + 2
					continue
				}
			}

			result = append(result, content[pos:n])
			pos = n
			i = n
		}
	}

	if pos < contentL-1 {
		result = append(result, content[pos:])
	}
	return
}
