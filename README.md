## V2

#### 使用
```
./linux-server -h

>>>>>
GPT接口适配器。统一适配接口规范，集成了bing、claude-2，gemini...
项目地址：https://github.com/bincooo/chatgpt-adapter

Usage:
  ChatGPT-Adapter [flags]

Flags:
  -h, --help             help for ChatGPT-Adapter
      --port int         服务端口 port (default 8080)
      --proxies string   本地代理 proxies
  -v, --version          version for ChatGPT-Adapter
```


启动服务，如果网络不在服务区域，请尝试设置/替换 `proxies`

```
./linux-server --port 8080 --proxies socks5://127.0.0.1:7890
```

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
> 在 `www.coze.com` 官网中登陆，浏览器 `cookies` 中取出 `sessionid` + `msToken` 的值就是 `Authorization` 参数
> 
> 格式例子:
> 781xxx[msToken=xxx]


#### free画图接口

提供了 `coze.dall-e-3`、 `sd.dall-e-3`、`xl.dall-e-3`, `kb.dall-e-3`, `pg.dall-e-3`，它们会根据你提供的 `Authorization` 选择其中的一个

```txt
// 下面三个固定写法

// sd.dall-e-3
Authorization: sk-prodia-sd

// xl.dall-e-3
Authorization: sk-prodia-xl

// kb.dall-e-3
Authorization: sk-krebzonide
```

api:

```txt
/v1/chat/generations
/v1/object/generations
/proxies/v1/chat/generations
```

```curl
curl -i -X POST \
   -H "Content-Type:application/json" \
   -H "Authorization: xxx" \
   -d \
'{
  "prompt":"一个二次元少女",
  "style":"",
  "model":"dall-e-3",
  "n":1
}' \
 'http://127.0.0.1:8080/v1/chat/generations'
```

#### 特殊标记增强

[flags](flags.md)