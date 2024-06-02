![Screenshot 2024-04-18 at 04 03 41](https://github.com/bincooo/chatgpt-adapter/assets/36452456/b130375c-f40b-404a-bade-6640f2aa29c9)

------------------------------------

<p align="center">
  <h2 align="center">Adapter for ChatGPT</h2>
  <p align="center">
    一款将免费服务整合到一起的ChatGPT接口服务！<br />
    *添加实验性toolCall能力，尝试让没有toolCall能力的AI也能执行任务*
  </p>
</p>

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
[
    {
        "id":       "claude",
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
        "id":       "gemini-1.0",
        "object":   "model",
        "created":  1686935002,
        "owned_by": "gemini-adapter"
    },
    {
        "id":       "command-r-plus",
        "object":   "model",
        "created":  1686935002,
        "owned_by": "cohere-adapter"
    }
    (更多模型请访问API获取) ...
]
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

<details>
<summary> 效果预览 1 </summary>

  - LobeChat
<pre>
    <img width="451" alt="Screenshot 2024-05-19 at 01 53 05" src="https://github.com/bincooo/chatgpt-adapter/assets/36452456/e055af22-38c4-4a05-bc1b-9f5e9e89beeb">
</pre>
</details>
<details>
<summary> 效果预览 2 </summary>

  - FastGPT
<pre>
    <img width="451" alt="Screenshot 2024-05-19 at 01 54 26" src="https://github.com/bincooo/chatgpt-adapter/assets/36452456/a41a15c2-5d81-4029-ad43-72ac7e92e93c">
</pre>
</details>
<details>
<summary> 效果预览 3 </summary>

  - google模型原生toolCall运行良好，其它皆为提示词实现toolCall。

  - 若想达到多个工具执行效果，请开启 < tool tasks />。
<pre>
<img width="451" alt="Screenshot 2024-05-23 at 03 13 09" src="https://github.com/bincooo/chatgpt-adapter/assets/36452456/faa16d95-a082-4e90-826e-73b7055fad8f">
<img width="451" alt="Screenshot 2024-05-23 at 03 21 34" src="https://github.com/bincooo/chatgpt-adapter/assets/36452456/a59cfba6-11b7-419e-bb3e-84d28c018fbd">
<img width="451" alt="Screenshot 2024-05-23 at 03 30 29" src="https://github.com/bincooo/chatgpt-adapter/assets/36452456/baa0020c-1da3-4302-8705-8d8abdbbff97">
</pre>
</details>

#### Authorization 获取

claude:
> 在 `claude.ai` 官网中登陆，浏览器 `cookies` 中取出 `sessionKey` 的值就是 `Authorization` 参数

bing:
> 在 `www.bing.com` 官网中登陆，浏览器 `cookies` 中取出 `_U` 的值就是 `Authorization` 参数

gemini:
> 在 `ai.google.dev` 中申请，获取 token凭证就是 `Authorization` 参数

coze:
> 在 `www.coze.com` 官网中登陆，浏览器 `cookies` 中复制完整的 `cookie` 就是 `Authorization` 参数

> 支持指定bot模型：
> 
> 格式 -> coze/botId-version-scene;
> 例子 -> coze/7353052833752694791-1712016747307-2
> 
> 支持开发者模式，该模式下可修改全局变量TopP、Temperature、MaxTokens。
> 但是会出现排队情况，建议多账号轮询使用
>
> cookie需为botId自己所属的账号， 结尾 o 固定
> 
> 格式 -> coze/botId-spaceId-scene-o; 
> 例子 -> coze/7353052833752694791-xxx-4-o

lmsys:
> 无需cookie， model参数为 `lmsys/` 前缀，例：`lmsys/claude-3-haiku-20240307`
> 该接口有第三方监管，但用来进行正向对话还是不错的。对ip严苛

custom:
> 实现chatgpt规范的上游AI接口，可用此定义来实现toolCall的功能
> 在原模型的名称前添加： `custom/` 前缀，例：`custom/freeGpt35`
> 
> 而后在 `role`为 `user`、`system` 中的一个里添加 `<tool enabled />` 即可开启toolCall
> 
> 需在 `config.yaml` 里配置 `custom-llm` 属性

#### free画图接口

提供了 `coze.dall-e-3`、 `sd.dall-e-3`、`xl.dall-e-3`、 `pg.dall-e-3`、 `google.dall-e-3`，它们会根据你提供的 `Authorization` 选择其中的一个

```txt
// 下面固定写法

// sd.dall-e-3
Authorization: sk-prodia-sd

// xl.dall-e-3
Authorization: sk-prodia-xl

// dalle-4k.dall-e-3
Authorization: sk-dalle-4k

// google.dall-e-3
Authorization: sk-google-xl
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

### 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=bincooo/chatgpt-adapter&type=Date)](https://star-history.com/#bincooo/chatgpt-adapter&Date)
