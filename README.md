## V2

#### 请求列表

model 列表
```txt
{
    "id":       "claude-2",
    "object":   "model",
    "created":  1686935002,
    "owned_by": "claude-adapter"
},
{
    "id":       "bing",
    "object":   "model",
    "created":  1686935002,
    "owned_by": "bing-adapter"
},
{
    "id":       "coze",
    "object":   "model",
    "created":  1686935002,
    "owned_by": "coze-adapter"
},
{
    "id":       "gemini",
    "object":   "model",
    "created":  1686935002,
    "owned_by": "gemini-adapter"
}
```

completions 对话
```txt
/v1/chat/completions
/v1/object/completions
/proxies/v1/chat/completions
```

```curl
curl -i -X POST \
   -H "Content-Type:application/json" \
   -H "Authorization: xxx" \
   -d \
'{
  "stream": true,
  "model": "coze",
  "messages": [
    {
      "role":    "user",
      "content": "hi"
    }
  ]
}' \
 'http://127.0.0.1:8080/v1/chat/completions'
```


#### Authorization 获取

claude:
> 在 `claude.ai` 官网中登陆，浏览器 `cookies` 中取出 `sessionKey` 的值就是 `Authorization` 参数

bing:
> 在 `www.bing.com` 官网中登陆，浏览器 `cookies` 中取出 `_U` 的值就是 `Authorization` 参数

gemini:
> 在 `ai.google.dev` 中申请，获取 token凭证就是 `Authorization` 参数

coze:
> 在 `www.coze.com` 官网中登陆，浏览器 `cookies` 中取出 `sessionid` 、`msToken` 的值就是 `Authorization` 参数
>
> 格式拼接： 
> 
> ${sessionid}[msToken=${msToken}]
> 
> 例子：
> 
> 3fdb9fb39a9bc013049e4315c5xxx[msToken=xxx]