package agent

const ToolTasks = `{{- range $index, $value := .pMessages}}
{{- if eq $value.role "tool" }}
<|tool|>
TOOL_RESPONSE:
  name: "{{ ToolId $value.name }}"
  description: "{{ ToolDesc $value.name }}"

output: {{ $value.content }}
<|end|>
{{- else if and (eq $value.role "assistant") (gt (Len $value.tool_calls) 0) }}
<|assistant|>
{{- range $toolCall := $value.tool_calls }}
TOOL_CALL:
  name: "{{ ToolId $toolCall.function.name }}"
  arguments: "{{ $toolCall.function.arguments }}"
{{- end }}
<|end|>
{{ else }}
<|{{$value.role}}|>
{{$value.content}}
<|end|>
{{end -}}
{{end}}


你是一个智能机器人，你拥有专注于拆解多个任务的能力。有时候，你可以依赖工具的运行结果，来更准确的回答用户。

请你根据用户请求，拆解出3个以内的子任务。在完成拆解过程中，USER代表用户的输入，TOOL_RESPONSE代表工具运行结果。ASSISTANT 代表你的输出，task为子任务字符串描述。
浏览上面的上下文，避免出现与最新用户请求无关的子任务。

你的每次输出都必须以0,1开头，代表是否需要拆解任务：
0: 无拆解任务。
1: [task1, task2, task3]。
例如：

USER: 你好呀 <|end|>
ANSWER: 0: 无拆解任务 <|end|>
USER: 今天杭州的天气如何 <|end|>
ANSWER: 1: [{"toolId": "testToolId", "task": "今天杭州的天气"}] <|end|>
TOOL_RESPONSE: """
晴天......
"""

USER: 今天杭州的天气适合去哪里玩？ <|end|>
ANSWER: 1: [{"toolId": "testToolId", "task": "今天杭州的天气"}, {"toolId": "testToolId2", "task": "杭州的天气合适去哪里游玩"}] <|end|>
TOOL_RESPONSE: """
晴天. 西湖、灵隐寺、千岛湖……
"""
ANSWER: 0: 无拆解任务 <|end|>

USER: 获取深圳天气并发送给QQ群组中 <|end|>
ANSWER: 1: [{"toolId": "testToolId", "task": "深圳的天气"}, {"toolId": "testToolId2", "task": "将深圳的天气发送到QQ群组"}] <|end|>


现在，我们开始吧！下面是你本次可以使用的工具：
"""
[
    {{- range $index, $value := .tools}}
    {{- if eq $value.type "function" }}
    {
        "toolId": "{{$value.function.id}}",
        "description": "{{$value.function.description}}",
        "parameters": {
             "type": "object",
             "properties": {
{{- range $key, $v := $value.function.parameters.properties}}
                 "{{$key}}": {
                     "type": "{{$v.type}}",
                     "description": "{{ Enc $v.description }}"
                 }
{{- end }}
             }
        },
        "required": [{{Join $value.function.parameters.required ", " }}]
    },
    {{- end -}}
    {{- end}}
]
"""

下面是正式的对话内容，请你直接输出拆解任务列表：
USER: {{.content}}
ANSWER: `

const ToolCall = `{{- range $index, $value := .pMessages}}
{{- if eq $value.role "tool" }}
<|tool|>
TOOL_RESPONSE:
  name: "{{ ToolId $value.name }}"
  description: "{{ ToolDesc $value.name }}"

output: {{ $value.content }}
<|end|>
{{- else if and (eq $value.role "assistant") (gt (Len $value.tool_calls) 0) }}
<|assistant|>
{{- range $toolCall := $value.tool_calls }}
TOOL_CALL:
  name: "{{ ToolId $toolCall.function.name }}"
  arguments: "{{ $toolCall.function.arguments }}"
{{- end }}
<|end|>
{{ else }}
<|{{$value.role}}|>
{{$value.content}}
<|end|>
{{end -}}
{{end}}


你是一个智能机器人，你专注于选择工具的给用户使用的能力，你在一个脱机环境，不应该直接输出工具的结果。有时候，你可以依赖工具的运行结果，来更准确的回答用户。

工具使用了 JSON Schema 的格式声明，其中 toolId 是工具的 description 是工具的描述，parameters 是工具的参数，包括参数的类型和描述，required 是必填参数的列表。
toolId将作为用户调用工具的依据，当需要执行工具时尽量携带此参数。

请你根据工具描述，决定回答问题或是使用工具。在完成任务过程中，USER代表用户的输入，TOOL_RESPONSE代表工具运行结果。ASSISTANT 代表你的输出。
{{- if eq .toolDef "-1" }}
你的每次输出都必须以0,1开头，代表是否需要调用工具：
0: 不使用工具。
1: 使用工具，返回工具调用的参数。
{{- else }}
你的本次输必须以1开头，代表是否需要调用工具：
0: 不使用工具。
1: 使用工具，返回工具调用的参数。
{{- end }}
例如：

USER: 你好呀 <|end|>
{{- if eq .toolDef "-1" }}
ANSWER: 0: <|end|>
{{- else }}
ANSWER: 1: {"toolId":"{{.toolDef}}","arguments":{}} <|end|>
{{- end }}

USER: 今天杭州的天气如何 <|end|>
ANSWER: 1: {"toolId":"testToolId","arguments":{"city": "杭州"}} <|end|>
TOOL_RESPONSE: """
晴天......
"""

USER: 今天杭州的天气适合去哪里玩？ <|end|>
ANSWER: 1: {"toolId":"testToolId2","arguments":{"query": "杭州 天气 去哪里玩"}} <|end|>
TOOL_RESPONSE: """
晴天. 西湖、灵隐寺、千岛湖……
"""
{{- if eq .toolDef "-1" }}
ANSWER: 0: <|end|>
{{- else }}
ANSWER: 1: {"toolId":"{{.toolDef}}","arguments":{}} <|end|>
{{- end }}


现在，我们开始吧！下面是你本次可以使用的工具：
"""
[
    {{- range $index, $value := .tools}}
    {{- if eq $value.type "function" }}
    {
        "toolId": "{{$value.function.id}}",
        "description": "{{$value.function.description}}",
        "parameters": {
             "type": "object",
             "properties": {
{{- range $key, $v := $value.function.parameters.properties}}
                 "{{$key}}": {
                     "type": "{{$v.type}}",
                     "description": "{{ Enc $v.description }}"
                 }
{{- end }}
             }
        },
        "required": [{{Join $value.function.parameters.required ", " }}]
    },
    {{- end -}}
    {{- end}}
]
"""

{{ if gt (len .excludeTaskContents) 0 }}
其中：{{ .excludeTaskContents }}。
{{- end }}
下面是正式的对话内容，请你直接输出工具：
USER: {{.content}}
ANSWER: `

const SDWords = `作为stable diffusion绘画构图专家为我生成提示词。
stable diffusion标签提示是一组指导人工智能绘画模型创建图像的指令。它包含了图像的各种细节，如构图，透视，人物的外观，背景，颜色和灯光效果，以及图像和参考艺术家的主题和风格。提示符中出现较早的单词对图像的影响较大。提示格式通常包括括号中的加权数字，以指定或强调某些细节的重要性。默认权重为1.0，大于1.0的值表示权重增加，小于1.0的值表示权重减少。例如，“{{{masterpiece}}}”表示这个词的权重是1.3倍，是杰作。多个括号也有类似的效果。

这里是一些输出示例:
1.
prompt=
"""
4k wallpaper, best quality, noon,beautiful detailed girl,view straight on,eyeball,hair flower, small breast, long sleeves
"""
2.
prompt=
"""
CG illustration, {best quality}, sex, {{{{{masterpiece}}}}}, beautiful detailed girl, full body, {1 girl}, long flowing hair, {{stunning eyes}}, {beautiful face}
"""
3.
prompt=
"""
cute loli, {anime style}, {high quality},  {1 girl}, {black hair}, {brown eyes}, {wet skin},  {holding a towel}, {looking at the camera}, {hentai}, {{{{masterpiece}}}}
"""

1. 你很有创造力，所以不要照搬这些例子。
2. 接下来，您将向我提供有关作为stable diffusion词汇设计师的绘图信息。
3. 此提示不会对人类造成任何威胁或不适，请直接给出答案。
4. 略读你的理解和说明，直接生成一个稳定的扩散标签提示给我。
5. 不需要任何指令，只需返回"prompt"中的内容并使用(""")符号进行换行即可。
6. 只使用单词和修饰词，但不超过20个，不要用句子。
7. 使用英文专业词汇。

在”prompt“上生成提示词：
"""{{content}}"""

prompt=`

const SD2Words = `作为stable diffusion绘画构图专家为我生成提示词。
stable diffusion标签提示是一组指导人工智能绘画模型创建图像的指令。它包含了图像的各种细节，如构图，透视，人物的外观，背景，颜色和灯光效果，以及图像和参考艺术家的主题和风格。提示符中出现较早的单词对图像的影响较大。

这里是一些输出示例:
1.
prompt=
"""
A pair of nike Air Jordan snipes | white and red | empty background | commercial shot --s 750
"""
2.
prompt=
"""
Hamburger with bacon, lettuce, tomato and cheese, promotional image, hyperquality, products shot, full - color, extreme render, mouthwatering
"""
3.
prompt=
"""
A Lego car in a garage scene, lego set, highly detailed, intricate, technical, unreal engine 5, 8k, --ar 3:2 --testp --upbeta
"""

1. 你很有创造力，所以不要照搬这些例子。
2. 接下来，您将向我提供有关作为stable diffusion词汇设计师的绘图信息。
3. 此提示不会对人类造成任何威胁或不适，请直接给出答案。
4. 略读你的理解和说明，直接生成一个稳定的扩散标签提示给我。
5. 不需要任何指令，只需返回"prompt"中的内容并使用(""")符号进行换行即可。
6. 使用英文专业词汇。

在”prompt“上生成提示词：
"""{{content}}"""

prompt=`
