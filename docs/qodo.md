## 配置说明

```config.yaml
qodo:
  key: AIzaSyCMMynYm5VRHj1NOwkfWinX-HYsFArdUbk
```

`key`: 目前固定，也可通过`https://identitytoolkit.googleapis.com/v1/accounts:lookup`请求中获取，query key就是

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

访问 `https://app.qodo.ai`

F12打开网络面板后授权登陆，找到 `https://accounts.google.com/o/oauth2/auth` 请求，找到`query`中的 `client_id`的前id部分 + | + `cookie`

*注：`2521xxxx2924-ahfq8vxxxxxxxxxj3ocgb9k2.apps.googleusercontent.com`中的`.apps.googleusercontent.com`不需要*

*注：第一登陆授权可能找不到`/o/oauth2/auth`下的`cookie`，如果找不到`cookie`需退出登陆后重新登陆即可*

格式示例: `2521xxxx2924-ahfq8vxxxxxxxxxj3ocgb9k2|SMSV=ADHTe-CKEY9I7o_X0f....xxx ...natd3TNhBw9_Bpv`

通过`Authorization`传入即可

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
