## 配置说明

```config.yaml
coze:
  websdk:
    model: claude-35-sonnet-200k
    system: xxx
    bot: custom-assistant
    accounts:
      - email: xxx@gmail.com
        password: xxx
        validate: xxx@gmail.com
```

`accounts` 为gmail登陆邮箱，仅限通过输入邮箱验证登陆的账户可用，未实现人机验证。对网络要求较高

`model` 值可选：`claude-35-sonnet-200k`  `claude-35-haiku-200k`  `gpt4o-8k`  `gpt4o-32k`  `gpt4o-128k`  `gpt4-125k`  `gpt35-16k`

也可以自行部署在coze.com部署websdk后获取bot_id，然后拼接成 `coze/{bot_id}-xxxxx-1000-w`字符串。
`coze/{bot_id}-xxxxx-1000-w`便是你的 model 参数

## 模型列表

```json
[
    "coze/websdk",
    "coze/{bot_id}-xxxx-1000-w"
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
  "model": "coze/websdk",
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
