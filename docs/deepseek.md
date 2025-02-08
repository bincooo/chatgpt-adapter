**开发中 ...**

## 配置说明

```config.yaml
deepseek:
  userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15'
  cookie: 'intercom-device-id-gxxx .......... xxx'
```

`userAgent`: 浏览器userAgent

`cookie`: 浏览器cookie ，主要包含cf_clearance参数



中国地区不需要配置以上信息。

海外地区出现403可尝试配置，要求你在浏览器中捉取`userAgent`、`cookie`时与你搭建本服务的IP始终一致！否则过盾失败！！！



## 模型列表

```json
[
    "deepseek-chat",
    "deepseek-reasoner"
]
```

## 请求示例

`authorization`: 在浏览器中请求体携带的头部`Authorization`

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "stream": true,
  "model": "deepseek-chat",
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
