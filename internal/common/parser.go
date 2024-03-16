package common

import (
	"fmt"
	"regexp"
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
	attr    map[string]interface{}
	parent  *XmlNode
	child   []*XmlNode
}

type XmlParser struct {
	whiteList []string
}

// 只解析whiteList中的标签
func NewParser(whiteList []string) XmlParser {
	return XmlParser{whiteList}
}

// xml解析的简单实现
func (xml XmlParser) Parse(value string) []*XmlNode {
	messageL := len(value)
	if messageL == 0 {
		return nil
	}

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
			if i+l >= contentL {
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
	parseAttr := func(slice []string) map[string]any {
		attr := make(map[string]any)
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

			v1, err := strconv.Atoi(it[n+1:])
			if err == nil {
				attr[it[:n]] = v1
				continue
			}

			v2, err := strconv.ParseFloat(it[n+1:], 10)
			if err == nil {
				attr[it[:n]] = v2
				continue
			}

			v3, err := strconv.ParseBool(it[n+1:])
			if err == nil {
				attr[it[:n]] = v3
				continue
			}

			if it[n+1] == '"' && it[len(it)-1] == '"' {
				attr[it[:n]] = it[n+2 : len(it)-1]
			}
		}
		return attr
	}

	// =============
	// =============

	content := value
	contentL := len(content)
	type skv struct {
		s []*XmlNode
		p *skv
	}

	slice := &skv{make([]*XmlNode, 0), nil}

	var curr *XmlNode = nil
	for i := 0; i < contentL; i++ {
		if content[i] == '<' {
			// =========================================================
			// ⬇⬇⬇⬇⬇ 结束标记 ⬇⬇⬇⬇⬇
			if next(content, i, '/') {
				n := search(content, i, '>')
				// 找不到 ⬇⬇⬇⬇⬇
				if n == -1 {
					// 丢弃
					curr = nil
					break
				}

				if curr == nil {
					continue
				}
				// 找不到 ⬆⬆⬆⬆⬆

				split := strings.Split(curr.tag, " ")
				if split[0] == content[i+2:n] {
					step := 2 + len(curr.tag)
					curr.t = XML_TYPE_X
					curr.end = n + 1
					curr.content = content[curr.index+step : curr.end-len(split[0])-3]
					// 解析xml参数
					if len(split) > 1 {
						curr.tag = split[0]
						curr.attr = parseAttr(split[1:])
					}
					i = n

					slice.s = append(slice.s, curr)
					curr = curr.parent
					if curr != nil {
						curr.child = slice.s
					}
					if slice.p != nil {
						slice = slice.p
					}
				}
				// ⬆⬆⬆⬆⬆ 结束标记 ⬆⬆⬆⬆⬆

				// =========================================================
				//

			} else if nextStr(content, i, "!--") {

				//
				// ⬇⬇⬇⬇⬇ 是否是注释 <!-- xxx --> ⬇⬇⬇⬇⬇

				n := searchStr(content, i+3, "-->")
				if n < 0 {
					i += 2
					continue
				}

				slice.s = append(slice.s, &XmlNode{index: i, end: n + 3, content: content[i : n+3], t: XML_TYPE_I})
				i = n + 2
				// ⬆⬆⬆⬆⬆ 是否是注释 <!-- xxx --> ⬆⬆⬆⬆⬆

				// =========================================================
				//

			} else {

				//
				// ⬇⬇⬇⬇⬇ 新的 XML 标记 ⬇⬇⬇⬇⬇

				n := search(content, i, '>')
				if n == -1 {
					break
				}

				tag := content[i+1 : n]
				// whiteList 为nil放行所有标签，否则只解析whiteList中的
				contains := xml.whiteList == nil || ContainFor(xml.whiteList, func(item string) bool {
					if strings.HasPrefix(item, "reg:") {
						compile := regexp.MustCompile(item[4:])
						return compile.MatchString(tag)
					}
					return item == tag
				})

				if !contains {
					i = n
					continue
				}

				if curr == nil {
					curr = &XmlNode{index: i, tag: tag, t: XML_TYPE_S}
					//slice.s = append(slice.s, curr)
				} else {
					node := &XmlNode{index: i, tag: tag, t: XML_TYPE_S, parent: curr}
					slice = &skv{make([]*XmlNode, 0), slice}
					//slice.s = append(slice.s, node)
					curr = node
				}
				i = n
				// ⬆⬆⬆⬆⬆ 新的 XML 标记 ⬆⬆⬆⬆⬆
			}
		}
	}

	// =========================================================
	return slice.s
}

func XmlPlot(messages []map[string]string) {
	if len(messages) == 0 {
		return
	}

	inserts, regexs := xmlPlotToHandleContents(messages)

	// 正则替换
	for _, regex := range regexs {
		s := strings.Split(regex['v'], ":")
		if len(s) < 2 {
			continue
		}

		before := strings.TrimSpace(s[0])
		after := strings.TrimSpace(strings.Join(s[1:], ""))
		if before == "" {
			continue
		}

		c := regexp.MustCompile(before)
		for _, message := range messages {
			message["content"] = c.ReplaceAllString(message["content"], after)
		}
	}

	// 深度插入
	for _, insert := range inserts {
		i, _ := strconv.Atoi(insert['i'])
		messageL := len(messages)
		if insert['o'] == "true" && messageL-1 < Abs(i) {
			continue
		}

		pos := messageL - 1 - i
		if pos < 0 {
			pos = 0
		}
		if pos >= messageL {
			pos = messageL - 1
		}

		messages[pos]["content"] += "\n\n" + insert['v']
	}
}

// inserts: 深度插入, i 是深度索引，v 是插入内容， o 是指令
// regexs: v 是正则内容
func xmlPlotToHandleContents(messages []map[string]string) (inserts []map[uint8]string, regexs []map[uint8]string) {
	var (
		parser = NewParser([]string{
			"regex",
			`reg:@-*\d+`,
		})
	)

	for _, message := range messages {
		role := message["role"]
		if role != "assistant" && role != "system" {
			continue
		}

		clean := func(ctx string) {
			message["content"] = strings.Replace(message["content"], ctx, "", -1)
		}

		content := message["content"]
		nodes := parser.Parse(content)
		if len(nodes) == 0 {
			continue
		}

		for _, node := range nodes {
			// 注释内容删除
			if node.t == XML_TYPE_I {
				clean(content[node.index:node.end])
			}

			// 自由深度插入
			if node.t == XML_TYPE_X && node.tag[0] == '@' {
				c, _ := regexp.Compile(`@-*\d+`)
				if c.MatchString(node.tag) {
					// 消息上下文次数少于插入深度时，是否忽略
					// 如不忽略，将放置在头部或者尾部
					miss := "true"
					if node.attr != nil {
						if it, ok := node.attr["miss"]; ok {
							if v, o := it.(bool); !o || !v {
								miss = "false"
							}
						}
					}
					inserts = append(inserts, map[uint8]string{'i': node.tag[1:], 'v': node.content, 'o': miss})
					clean(content[node.index:node.end])
				}
			}

			// 正则替换
			if node.t == XML_TYPE_X && node.tag == "regex" {
				order := "0" // 优先级
				if o, ok := node.attr["order"]; ok {
					order = fmt.Sprintf("%v", o)
				}
				regexs = append(regexs, map[uint8]string{'o': order, 'v': node.content})
				clean(content[node.index:node.end])
			}
		}
	}

	if len(regexs) > 0 {
		sort.Slice(regexs, func(i, j int) bool {
			return regexs[i]['o'] > regexs[j]['o']
		})
	}
	return
}
