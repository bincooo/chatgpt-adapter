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

#### 注释，只为了标注说明，没有实际作用
```text
<!-- 我是注释 -->
```

#### cdata，避免属性/标签体中使用<>箭括号解析问题
```text
<![CDATA[ xxx ]]>
```

#### bing 模型 开启 notebook 模式
```text
flag: notebook

attribute:
    disabled: (bool) 是否禁用，默认false

<notebook />
<notebook disabled />
```

#### 正则替换
```text
flag: regex

attribute:
    order: (int) 优先级，值越大优先级越大
    miss: (int) 跳过 `messages` 行，从尾部开始往回数
```
例子：
```text
<regex miss=2>```((?!```).)+```:</regex>

>>>>>
效果是除了system对话和倒数的2行外，将所有的markdown 代码块删除

>>>>> 
{
  "messages": [
    {
      "content": "<regex miss=2>```((?!```).)+```:</regex> 你是一个猫娘xxx...",
      "role": "system"
    },
    {
      "content": "好的喵\n```\n年龄: 15\n身高: 150\n服饰: xxx```",
      "role": "assistant"
    },
    {
      "content": "研读书籍，准备明天的测验",
      "role": "user"
    },
    
    ... ...
  ],
  "model": "coze",
  "stream": false
}

最终效果
>>>>>
{
  "messages": [
    {
      "content": "你是一个猫娘xxx...",
      "role": "system"
    },
    {
      "content": "好的喵\n",
      "role": "assistant"
    },
    {
      "content": "研读书籍，准备明天的测验",
      "role": "user"
    },
    
    ... ...
  ],
  "model": "coze",
  "stream": false
}
```

#### 深度插入
```text
flag: @-*\d+    正数向下数，负数向上数

attribute:
    miss: (bool) 超出 `messages` 长度时是否忽略 true/false, 为false时会拼接到第一条或者最后一条
    role: (string) 留空或没有属性会在索引下的消息体末尾拼接，有值则在索引位置插入消息体:{ "role": role, "content": body }
```

例子：
```text
<@1 miss=true> hello ~ </@1>

>>>>> 
{
  "messages": [
    {
      "content": "<@1 miss=false> hello ~ </@1> 你是一个猫娘xxx...",
      "role": "system"
    },
    {
      "content": "好的喵\n```\n年龄: 15\n身高: 150\n服饰: xxx```",
      "role": "assistant"
    },
    {
      "content": "研读书籍，准备明天的测验",
      "role": "user"
    },

    ... ...
  ],
  "model": "coze",
  "stream": false
}

最终效果
>>>>>
{
  "messages": [
    {
      "content": "你是一个猫娘xxx...",
      "role": "system"
    },
    {
      "content": "好的喵\n```\n年龄: 15\n身高: 150\n服饰: xxx```",
      "role": "assistant"
    },
    {
      "content": "研读书籍，准备明天的测验\n\n hello ~ ",
      "role": "user"
    },

    ... ...
  ],
  "model": "coze",
  "stream": false
}
```


#### 流响应标记处理 (正则替换)
```text
flag: matcher

attribute:
    find: (int) 标记
    len: (int) 缓存长度
```

例子：
```text
<matcher find="<![CDATA[<img>]]>" len=30> <![CDATA[<img>(.+)</img>]]>:![image]($1) </matcher>

>>>>> 
{
  "messages": [
    {
      "content": "<matcher find="<![CDATA[<img>]]>" len=30> <![CDATA[<img>(.+)</img>]]>:![image](https://files.catbox.moe/$1) </matcher> 我说Hi，你回复<img>笑脸.jpg</img>",
      "role": "system"
    },
    {
      "content": "Hi",
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
      "content": "我说Hi，你回复<img>笑脸.jpg</img>",
      "role": "system"
    },
    {
      "content": "Hi",
      "role": "user"
    }
  ],
  "model": "coze",
  "stream": false
}

请求响应时：
{
    "id": "chatcmpl-completion",
    "object": "chat.completion",
    "created": 1710635684,
    "model": "coze",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "![image](https://files.catbox.moe/笑脸.jpg)",
                "tool_calls": null
            },
            "delta": null,
            "finish_reason": "stop"
        }
    ],
    "error": null
}
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
