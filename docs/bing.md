**开发中 ...**

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

*TIPS: 该 authorization 需要登陆后在F12网络面板处 `/common/oauth2/v2.0/token` 获取。*

格式：client_id | scope_id | refresh_token

例子：14638111-3389-403d-b206-a6a71d9fxxx|140e65af-45d1-4427-bf08-3e7295dxxx|M.C550_BAY.0.U.-ChfPypJYai2JPBV0wFJz075iLODHaxxxxxGZeFbEkYfWznXt0V5l0YDaXgBekptKYSuvOAcO*1wURFBpNpqK!kTyxU4jdENtPLuUaNEGKrDGPgU1ZJI9aQk7zs7yCcvEjRCldfMSH9CSzBXxeN6jc2kCz1gAI2rR92!S0DSvlZfJjQRupsXg0Zd3*O386hkne4or6sJkkeVz7VBTX13J7lb0S9SWU*j563PhVfv4Njt686Ghh*WSzvYlFkAQfuQBDPv16AjT9d*ISJtQC8jl*JE8GYWVuKeV!tIhFr89CfDWLpNkU3VzU4bVGfAh!JI8OYkoJ!XhcQWb88S3emtkJwk7VGYn5mS07PzDuR!IHqVh

控制台脚本一键获取：

```js
let token = '';
for (var i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (token == '' && key.includes('login.windows.net-accesstoken')) {
        const obj = JSON.parse(localStorage.getItem(key));
        token += obj.clientId + '|' + obj.target.split('/')[0];
    }
    if (token != '' && key.includes('login.windows.net-refreshtoken')) {
        const obj = JSON.parse(localStorage.getItem(key));
        token += '|' + obj.secret;
    }
}
console.log('token:', token);
```

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
