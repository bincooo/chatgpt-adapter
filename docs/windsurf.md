## 配置说明

```config.yaml
windsurf:
  proxied: false
```

`proxied`: 是否使用代理



## 模型列表

```json
[
    "windsurf/claude-3-5-sonnet",
    "windsurf/gpt4o",
    "windsurf/deepseek-chat",
    "windsurf/deepseek-reasoner",
    "windsurf/gpt4-o3-mini",
    "windsurf/gemini-2.0-flash",
]
```

## 请求示例

*TIPS: authorization 为[网页](https://codeium.com/profile)登陆后
访问 https://codeium.com/profile?referrer=extension F12 查看网络面版
的 https://web-backend.codeium.com/exa.user_analytics_pb.UserAnalyticsService/GetAnalytics 请求头x-api-key*

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "stream": true,
  "model": "windsurf/gpt4o",
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
    MaxTokens,
    TopK,
    TopP,
    
}
```
