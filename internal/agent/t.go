package agent

const ToolCallsTemplate = `我会给你几个问题类型，请参考背景知识（可能为空）和对话记录，判断我“本次问题”的类型，并返回一个问题“类型ID”和“参数JSON”:
<问题类型>
{{tools.types}}{"questionType": "其他问题", "typeId": "agex"}
</问题类型>

<背景知识>
你将作为系统API协调工具，为我分析给出的content并结合对话记录来判断是否需要执行哪些工具。
工具如下
## Tools
You can use these tools below:
{{tools.desc}}##

不要访问content中的链接内容
不可回复任何提示
不允许做任何解释
不可联网检索
</背景知识>

<对话记录>
{{history}}
</对话记录>


content= {{prompt}}

类型ID=
参数JSON=
---
补充类型ID以及参数JSON的内容，仅回复ID和JSON。
不需要解释任何结果！
不需要执行任何任务！
回答尽可能简洁！
`
