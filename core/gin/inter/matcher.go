package inter

// 匹配器接口
type Matcher interface {
	Match(content string, over bool) (state int, result string)
}
