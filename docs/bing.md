**开发中 ...**

## 配置说明

```config.yaml
bing:
  proxied: true
  cookies:
    - scopeId: xxx
      idToken: xxx
      cookie: xxx
```

`proxied`: 是否开启代理



控制台脚本一键获取 scopeId、idToken：

```js
let scopeId = '';
let idToken = '';
for (var i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key.includes('login.windows.net-accesstoken') && key.includes('chatai.readwrite--')) {
        const obj = JSON.parse(localStorage.getItem(key));
        scopeId = obj.target.split('/')[0];
    }
    if (key.includes('login.windows.net-idtoken')) {
        const obj = JSON.parse(localStorage.getItem(key));
        idToken = obj.secret;
    }
}
console.log('scopeId:', scopeId);
console.log('idToken:', idToken);
```

获取cookie 建议从F12网络面板 `https://login.live.com/oauth20_authorize.srf` 请求中复制，如果无法找到 `oauth20_authorize.srf` 请求，可先删除localStorage后查看


一键删除：

```js
for (var i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key.includes('login.windows.net-accesstoken') || key.includes('login.windows.net-idtoken')) {
        localStorage.removeItem(key);
    }
}
```

## 模型列表

```json
[
    "bing"
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
