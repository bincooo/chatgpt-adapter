## 配置说明

```config.yaml
qodo:
  mapC:
#    "xxx": "xxx"
```



## 模型列表

```json
[
    "qodo/claude-3-5-sonnet",
    "qodo/gpt-4o",
    "qodo/o1",
    "qodo/o3-mini",
    "qodo/o3-mini-high",
    "qodo/gemini-2.0-flash",
    "qodo/deepseek-r1",
    "qodo/deepseek-r1-full",
]
```

## 请求示例

F12打开网络面板找到`https://securetoken.googleapis.com/v1/token`请求，找到`query`中的 `key` + | + `refresh_token`



格式示例: `AIzaSy....-HxxxFArdUbk|AMf-vBx......I0hw5wnatd3TNhBw9_Bpv`

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "stream": true,
  "model": "qodo/gpt-4o",
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
