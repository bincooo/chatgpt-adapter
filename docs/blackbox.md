## 配置说明

```config.yaml
blackbox:
  token: '00f37b34-a166-4efb-bce5-1312d87f2f94'
  model:
    - 'xxx'
```

*TIPS: 该key目前固定*

## 模型列表

```json
[
    "blackbox/GPT-4o",
    "blackbox/Gemini-PRO",
    "blackbox/Claude-Sonnet-3.5"
]
```

## 请求示例

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "stream": true,
  "model": "blackbox/GPT-4o",
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
{
    "temperature": [0.0~1.0],
    "max_tokens": [0.0~1.0],
    "top_p": [0.0~1.0]
}
```
