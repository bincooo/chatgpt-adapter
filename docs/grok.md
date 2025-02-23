## 配置说明

```config.yaml
grok:
  think_reason: false
  disable_search: false
  cookies:
    - 'xxx'
```

`think_reason`: 是否开启思考模式

`disable_search`: 是否关闭联网搜索

`cookies`: cookie池



## 模型列表

```json
[
    "grok-2",
    "grok-3",
]
```

## 请求示例

F12打开网络面板抓取cookie即可, `author=cookie`

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "stream": true,
  "model": "grok-3",
  "messages": [
    {
      "role":    "user",
      "content": "hi ~"
    }
  ]
}' \
 'http://127.0.0.1:8080/v1/chat/completions'
```

可用参数：

```json
无
```
