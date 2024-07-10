package vars

var (
	Proxies = ""
)

const (
	MatDefault  int = iota // 执行下一个匹配器
	MatMatching            // 匹配中, 字符被缓存
	MatMatched             // 匹配器命中，不再执行下一个

	GinCompletion      = "__completion__"
	GinGeneration      = "__generation__"
	GinMatchers        = "__matchers__"
	GinCompletionUsage = "__completion-usage__"
	GinDebugger        = "__debug__"
	GinEcho            = "__echo__"
	GinTool            = "__tool__"
	GinClose           = "__close__"
	GinCharSequences   = "__char_sequences__"
	GinCancelFunc      = "__cancelFunc__"
	GinClaudeMessages  = "__claude_messages__"
)
