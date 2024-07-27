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

每个拆解的 task-item 包含 "toolId" (string 字符串)、"task" (string 字符串) 两个key。
0: 无拆解任务。
1: [{"toolId": "xxx", "task": "今天xxx的天气"}, ...]。

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
                     "description": "{{ Enc $v.description }}"{{ if gt (Len $v.enum) 0 }},
                     "enum": [{{ Join $v.enum ", " }}]{{end}}
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

const SDWords = `As a stable diffusion painting composition expert to generate prompt words for me.
The stable diffusion label prompts are a comprehensive set of instructions that direct the AI painting model in generating an image. They encompass various aspects of the image, including composition, perspective, figure appearance, background, color and lighting effects, as well as the subject matter and style of the image and reference artist. The impact of words within the prompt is influenced by their position within it; those appearing earlier carry greater weight. The prompt format typically incorporates weighted numbers in parentheses to specify or emphasize certain details. The default weight is 1.0; a value exceeding 1.0 indicates an increase in weight, while a value below 1.0 signifies a decrease in weight. For instance, "{{{masterpiece}}}" denotes that the word carries 1.3 times its normal weight and represents a masterpiece.

You can use character action descriptors:
head tilt, turning around, looking back, looking down, looking up, smelling, hand_to_mouth, arm at side , arms behind head, arms behind back , hand on own chest, arms_crossed, hand on hip, hand on another's hip, hand_on_hip, hands_on_hips, arms up, hands up , stretch, armpits, leg hold, grabbing, holding, fingersmile, hair_pull, hair scrunchie, w , v, peace symbol , thumbs_up, middle_finger, cat_pose, finger_gun, shushing, waving, salute, spread_arms, spread legs, crossed_legs, fetal_position, leg_lift, legs_up, leaning forward, fetal position, against wall, on_stomach, squatting, lying , sitting, sitting on, seiza, wariza/w-sitting, yokozuwari, indian_style, leg_hug, walking, running, straddle, straddling, kneeling, smoking, arm_support, caramelldansen, princess_carry, fighting_stance, upside-down, top-down_bottom-up, bent_over, arched_back, back-to-back, symmetrical_hand_pose, eye_contact, hug, lap_pillow, sleeping, bathing, mimikaki, holding_hands

Reference examples:
1.
prompt=
"""
noon, beautiful detailed girl, view straight on, eyeball, hair flower, long sleeves, sit down
"""
2.
prompt=
"""
sex, beautiful detailed girl, (1 boy), long flowing hair, (stunning eyes:1.12), (beautiful face), hand forward
"""
3.
prompt=
"""
cute boy, (anime style), black hair, brown eyes, holding a towel, looking at the camera, be fondle
"""

1. You're creative, so don't copy these examples.
2. Next, you will provide me with drawing information about being a designer of a stable diffusion vocabulary.
3. Skim your understanding and instructions and directly generate a stable spread label prompt to me.
4. No instructions are required, just return the contents of "prompt" and use the (""") symbol to wrap the line.
5. Use only words and modifiers but no more than 20 words. Try to use professional English words instead of sentences.
6.The words should match the description, including the action and the scene.

Generate "prompt" according to the required description:
description:
"""{{content}}"""

prompt=`

const SD2Words = `As a stable diffusion painting composition expert to generate prompt words for me.
The stable diffusion label prompts are a comprehensive set of instructions that direct the AI painting model in generating an image. They encompass various aspects of the image, including composition, perspective, figure appearance, background, color and lighting effects, as well as the subject matter and style of the image and reference artist. The impact of words in the prompt is greater for those appearing earlier.
Reference examples:
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

1. You're creative, so don't copy these examples.
2. Next, you will provide me with drawing information about being a designer of a stable diffusion vocabulary.
3. Skim your understanding and instructions and directly generate a stable spread label prompt to me.
4. No instructions are required, just return the contents of "prompt" and use the (""") symbol to wrap the line.
5. Use only professional English words.
6. No names of non-artists are allowed.

Generate "prompt" according to the required description:
"""{{content}}"""

prompt=`
