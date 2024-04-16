package common

import (
	"testing"
)

func TestParse(t *testing.T) {
	content := "111<!-- hello --><@-1>2<debug />22<regex> <![CDATA[<@-1>]]> </regex>"
	parser := XmlParser{
		[]string{
			"regex",
			`r:@-*\d+`,
			"debug",
			"matcher",
			"pad",      // bing中使用的标记：填充引导对话，尝试避免道歉
			"notebook", // notebook模式
			"histories",
			"cmd",
		},
	}

	nodes := parser.Parse(content)
	t.Log(nodes)
}
