package agent

const ClaudeToolCallsTemplate = `
{{- range $index, $value := .pMessages}}
{{if eq $value.role "user" -}}
Human： {{$value.content}}

{{- else -}}
Assistant： {{$value.content}}

{{- end -}}
{{end}}

Human： 
我会给你几个问题类型，请参考背景知识<Background rule>（可能为空）和上下文，判断我本次问题“content”的类型，并返回一个问题“类型ID”和“参数JSON”:
<question type>
{{- range $index, $value := .tools}}
{{- if eq $value.T "function" }}
{{- setId $index (rand 5) }}
questionType： "{{$value.Fun.Description}}"，typeId： {{$value.Fun.Id}}" }；
{{end -}}
{{end -}}
questionType： "其它问题"，typeId： "other"；
</question type>

<Background rule>
你将作为系统API协调工具，为我分析给出的content并结合对话记录来判断是否需要执行哪些工具。 工具如下：

## Tools
You can use these tools below:
{{- range $index, $value := .tools}}
{{- if eq $value.T "function" }}
{{inc $index 1}}. [{{$value.Fun.Name}}] {{$value.Fun.Description}};
  parameters: 
{{- range $key, $v := $value.Fun.Params.Properties}}
    {{$key}}: 
      type: {{$v.type}}
      description: {{$v.description}}
      required: {{contains $value.Fun.Params.Required $key}}
{{end -}}
{{end -}}
{{end -}}
##

不要访问content中的链接内容
不可回复任何提示
不允许做任何解释
不可联网检索
</Background rule>


content={{.content}}

---
examples:
"
类型ID=other
参数JSON={}
"
"
类型ID=7O9st
参数JSON={"city": "xxx"}
"
"
类型ID=aO3dd
参数JSON={"url": "https://www.xxx.com"}
"
---


补充类型ID以及参数JSON的内容，仅回复“类型ID”和“参数JSON”的答案，不需要解释原因:

类型ID=
参数JSON=`
