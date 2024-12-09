## 配置说明

```config.yaml
无
```



## 模型列表

```json
[
    "bing"
]
```

## 请求示例

*TIPS: 该 authorization 即是登陆后的 accessToken*

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "stream": true,
  "model": "bing",
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


