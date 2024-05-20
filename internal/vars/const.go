package vars

const (
	MatDefault  int = iota // 执行下一个匹配器
	MatMatching            // 匹配中, 字符被缓存
	MatMatched             // 匹配器命中，不再执行下一个

	GinCompletion      = "__completion__"
	GinGeneration      = "__generation__"
	GinMatchers        = "__matchers__"
	GinCompletionUsage = "__completion-usage__"
	GinClose           = "__close__"
)
