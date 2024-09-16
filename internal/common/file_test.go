package common

import (
	regexp "github.com/dlclark/regexp2"
	"testing"
)

func Test_download(t *testing.T) {
	file, err := Download(nil, "http://127.0.0.1:7890", "https://krebzonide-sdxl-turbo-with-refiner.hf.space/file=/tmp/gradio/e8cec4458822c7cd2308e8d36949cd3c1c446196/image.png", "png", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(file)
}

// (?=exp)：零宽度正预测先行断言，它断言自身出现的位置的后面能匹配表达式 exp
// (?<=exp)：零宽度正回顾后发断言，它断言自身出现的位置的前面能匹配表达式 exp
// (?!exp)：零宽度负预测先行断言，断言此位置的后面不能匹配表达式 exp
// (?<!exp)：零宽度负回顾后发断言来断言此位置的前面不能匹配表达式 exp
func Test_regex(t *testing.T) {
	var content = "欢迎回到豆之RPG游戏世界，鲁迪斯！你选择了一个充满神秘与古老魔法的中世纪世界。请稍等，正在为你构建这个世界...\n\n```\n正在生成中世纪魔法世界...\n\n██████████████] 99%\n\n配置环境变量...\n设定地理与历史背景...\n种族与文明构建中...\n魔法与技能系统初始化...\n```\n\n**生成完毕！**\n\n你所在的世界被称为安索瑞亚，一个充满魔法与万千种族的广袤大陆。中世纪的背景赋予了这片大陆深厚的历史感，既有壮丽的城堡，也有荒野中遗失的古老遗迹。魔法在这里是一切文明发展的基石，人们依靠它来照明、治疗疾病、构建建筑，甚至用于战斗与日常生活中的各种事务。\n\n你，鲁迪斯，是一位朝阳魔法学院的学徒，正值青涩年华。你有着一头乌黑的短发，双眸如同深邃的星空，身穿一袭简单的蓝色法袍，腰间别着一个小巧的材料袋，里面装着一些基础的魔法材料和你的魔法书。你渴望探索这个世界，学习更多的魔法知识，成为一名伟大的法师。\n\n**游戏开始！**\n\n你在朝阳学院的图书馆里，周围堆满了各式各样的书籍，从古老的魔法典籍到关于异兽的研究报告应有尽有。图书馆里弥漫着一种宁静祥和的气氛，偶尔传来几声学徒们低语讨论的声音。你今天的任务是研读一本关于基础防御魔法的书籍，以准备明天的测验。\n\n*你抬头望向窗外，窗外是一片宁静的景色，但远处的天际线上似乎有着暗云密布，这预示着将有一场风暴即将来临。*\n\n``` \n┌──═━┈ 世界线定位档案:\n⑆ 时间:[下午2点，多云转阴]\n⑆ 装备:[蓝色法袍，小巧的材料袋(含基础魔法材料和魔法书)]\n⑆ 任务:[研读基础防御魔法书籍，准备明天的测验]\n⑆ 环境:[朝阳魔法学院图书馆，静谧的学习环境]\n⑆ 伙伴:[暂无]\n⑆ 财富:[学徒津贴 - 15银币]\n⑆ 荣誉:[0 (初始值)]\n❤️ 健康:100/100\n✨ Buff: 无\n❌ Debuff: 无\n└──═━┈ \n✎[Ongoing effects, info] ... \n```\n\n*现在，你将如何行动？*\n\n```\n---\n请选择你的下一步行动：\n1️⃣ 研读书籍，准备明天的测验\n2️⃣ 向窗外的天际线走去，看个究竟\n3️⃣ 查找有关即将到来的风暴的资料\n4️⃣ 与其他学徒交流魔法知识\n```"
	//content = strings.ReplaceAll(content, "\n", "!u+000d!")
	var cmp = "```((?!```).)+```"
	c := regexp.MustCompile(cmp, regexp.Compiled)
	matched, err := c.Replace(content, "", -1, -1)
	//matched = strings.ReplaceAll(matched, "!u+000d!", "\n")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(matched)

	///

	content = "<![CDATA[111]]>222<![CDATA[333]]>"
	cmp = "<!\\[CDATA\\[(((?!]]>).)*)]]>"
	c = regexp.MustCompile(cmp, regexp.Compiled)
	matched, err = c.Replace(content, "$1", -1, -1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(matched)
}

func TestHashString(t *testing.T) {
	t.Log(HashString("你好，哔哩哔哩哔"))
}
