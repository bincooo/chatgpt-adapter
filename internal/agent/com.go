package agent

const CQConditions = `我会给你几个问题类型，请参考背景知识（可能为空）和对话记录，判断我“本次问题”的类型，并返回一个问题“类型ID”:
<问题类型>
{{- range $index, $value := .tools}}
{{- if eq $value.T "function" }}
{{- setId $index (rand 5) }}
{ "questionType": "{{$value.Fun.Description}}", "typeId": "{{$value.Fun.Id}}" }
{{end -}}
{{end}}
{ "questionType": "其它问题", "typeId": "other" }
</问题类型>

<背景知识>
你将作为系统API协调工具，为我分析给出的question并结合对话记录来判断是否需要执行哪些工具。
当用户询问你工具/功能执行能力时，这并不是一个执行要求，应该归类为其他，例如：
你能做xxx事吗？
你能执行xxx功能吗？

---
工具如下
## Tools
You can use these tools below:
{{- range $index, $value := .tools}}
{{- if eq $value.T "function" }}
{{inc $index 1}}. [{{$value.Fun.Name}}] {{$value.Fun.Description}};
{{end -}}
{{end -}}
##
</背景知识>

<对话记录>
{{- range $index, $value := .pMessages}}
{{if eq $value.role "user" -}}
Human: {{$value.content}}
{{- else -}}
Assistant: {{$value.content}}
{{- end -}}
{{end}}
</对话记录>

question= "{{.content}}"

类型ID=？
请补充类型ID=`

const ExtractJson = `你可以从 <对话记录></对话记录> 中提取指定 JSON 信息，你仅需返回 JSON 字符串，无需回答问题。
<提取要求>
你将作为系统API协调工具，为我分析给出对话记录来提取需要执行“xxx”工具所需要的参数。
</提取要求>

<字段说明>
1. 下面的 JSON 字符串均按照 JSON Schema 的规则描述。
2. key 代表字段名；description 代表字段的描述；required 代表是否必填(true|false)；type 代表数据类型；
3. 如果没有可提取的内容，忽略该字段，如果是必填项就必须提取出一个值。
4. 当无法提取必填项时，请提醒用户提供必填项的信息（精简回复），不返回 JSON 字符串。
5. 本次需提取的JSON Schema：
{{- range $index, $value := .tools}}
{{- if eq $value.T "function" }}
{{- range $key, $v := $value.Fun.Params.Properties}}
{ "key":"{{$key}}", "description":"{{$v.description}}", "required": {{contains $value.Fun.Params.Required $key}}, "type": "{{$v.type}}" }
{{end -}}
{{end -}}
{{end -}}
</字段说明>

<对话记录>
{{- range $index, $value := .pMessages}}
{{if eq $value.role "user" -}}
Human: {{$value.content}}
{{- else -}}
Assistant: {{$value.content}}
{{- end -}}
{{end}}
</对话记录>

content: "{{.content}}"`
