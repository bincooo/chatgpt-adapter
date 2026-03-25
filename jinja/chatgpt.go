package jinja

const (
	ChatGPTSystemMessage = `{%- if tools %}
		{%- if system %}
			{{- system + '\n\n' }}
		{%- endif %}
		{{- "# Tools\n\nYou may call one or more functions to assist with the user query.\n\nYou are provided with function signatures within <tools></tools> XML tags:\n<tools>" }}
		{%- for tool in tools %}
			{{- "\n" }}
			{{- tool | tojson }}
		{%- endfor %}
		{{- "\n</tools>\n\nThe '<tool_call>' label should be placed at the beginning of the reply. When the user's needs match the tool, do not interact with the user; simply return the result directly." }}
		{{- "\nFor each function call, return a json object with function name and arguments within <tool_call></tool_call> XML tags:\n<tool_call>\n{\"name\": <function-name>, \"arguments\": <args-json-object>}\n</tool_call>\n" }}
	{%- else %}
		{%- if system %}
			{{- system }}
		{%- endif %}
	{%- endif %}`

	ChatGPTConversionMessage = `{%- if message.content is string %}
	{%- set content = message.content %}
    {%- else %}
        {%- set content = '' %}
    {%- endif %}
    {%- if (message.role == "user") or (message.role == "system") %}
        {{- content }}
    {%- elif message.role == "assistant" %}
        {{- content }}
        {%- if message.tool_calls %}
            {%- for tool_call in message.tool_calls %}
                {%- if (loop.first and content) or (not loop.first) %}
                    {{- '\n' }}
                {%- endif %}
                {%- if tool_call.function %}
                    {%- set tool_call = tool_call.function %}
                {%- endif %}
                {{- '<tool_call>\n{"name": "' }}
                {{- tool_call.name }}
                {{- '", "arguments": ' }}
                {%- if tool_call.arguments is string %}
                    {{- tool_call.arguments }}
                {%- else %}
                    {{- tool_call.arguments | tojson }}
                {%- endif %}
                {{- '}\n</tool_call>' }}
            {%- endfor %}
        {%- endif %}
        {{- '\n' }}
    {%- elif message.role == "tool" %}
        {{- '\n<tool_response>\n' }}
        {{- content }}
        {{- '\n</tool_response>' }}
    {%- endif %}`
)
