增强标记，对话上下文预处理（预处理后这些标签不会遗留在上下文中）
#### 开启请求体打印
```text
flag: debug
```
例子：
```text
<debug />

>>>>> 
{
  "messages": [
    {
      "content": "<debug />\n研读书籍，准备明天的测验",
      "role": "user"
    }
  ],
  "model": "coze",
  "stream": false
}
```

#### 开启请求体响应：不消耗tokens，直接返回请求体
```text
flag: echo
```
例子：
```text
<echo />

>>>>> 
{
  "messages": [
    {
      "content": "<echo />\n研读书籍，准备明天的测验",
      "role": "user"
    }
  ],
  "model": "coze",
  "stream": false
}
```

#### cdata，避免属性/标签体中使用<>箭括号解析问题
```text
<![CDATA[ xxx ]]>
```

#### 开启 notebook 模式
```text
flag: notebook

attribute:
    disabled: (bool) 是否禁用，默认false

<notebook />
<notebook disabled />
```

#### 历史记录 (会放置到第一个user或assistant的前面)
```text
flag: histories
```

例子：
```text
<histories>[{"role": "user", "content": "hi!"}]</histories>

>>>>> 
{
  "messages": [
    {
      "content": "<histories>[{\"role\": \"assistant\", \"content\": \"了解，请问有什么可以帮到你？\"}]</histories> 你是一个拥有128k上下文token的gpt机器人",
      "role": "system"
    },
    {
      "content": "你好",
      "role": "user"
    }
  ],
  "model": "coze",
  "stream": false
}

最终效果
>>>>>
{
  "messages": [
    {
      "content": "你是一个拥有128k上下文token的gpt机器人",
      "role": "system"
    },
    {
      "role": "assistant",
      "content": "了解，请问有什么可以帮到你？"
    },
    {
      "content": "你好",
      "role": "user"
    }
  ],
  "model": "coze",
  "stream": false
}
```

#### tools 工具 开启 默认选中模式，作用是让工具选择在不匹配时默认选择一个，仅支持无参工具
```text
flag: tool

attribute:
    enabled: (bool) 是否开启toolCall，默认 false
    id: (string) 指定tool_function里的name值，默认-1
    tasks: (bool) 是否任务拆解，默认 false

使用示例
<tool enabled id="xxx" />
<tool enabled id="xxx" tasks />
```
