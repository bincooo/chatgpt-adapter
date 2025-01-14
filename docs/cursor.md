## 配置说明

```config.yaml
cursor:
  checksum: 'zo-v9_v2qxRYcau35NDnAHAVxQkLe6IHw8opkpKV4oLyo0PhPeSpj4QTw2VJ20Lngrz7XNTQ/clDRF5FOm3B1uK-mQDyFBRqD8JNj4kLByaAfm4AqK6IMbFYcrqXMMXexubsTRrr1'
```

*TIPS: 该key为设备id，随机。若不配置将自动生成一个*

## 模型列表

```json
[
    "cursor/claude-3-5-sonnet-20241022",
    "cursor/claude-3-opus",
    "cursor/claude-3.5-haiku",
    "cursor/claude-3.5-sonnet",
    "cursor/cursor-small",
    "cursor/gpt-3.5-turbo",
    "cursor/gpt-4",
    "cursor/gpt-4-turbo-2024-04-09",
    "cursor/gpt-4o",
    "cursor/gpt-4o-mini",
    "cursor/o1-mini",
    "cursor/o1-prevew"
]
```

## 请求示例

*TIPS: authorization 为[网页](https://www.cursor.com)登陆后的 cookie WorkosCursorSessionToken*

*TIPS: 目前cursor对设备码进行校验。诸位嫖友可以先从 [get-checksum](https://cc.wisdgod.com/get-checksum) 处获取设备码，在config.yaml中设置 *

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "stream": true,
  "model": "cursor/gpt-3.5-turbo",
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
