创建一个 `config.yaml`文件：

```config.yaml
server:
  port: 8080
  proxied: http://127.0.0.1:7890
  password: 'xxx'
  think_reason: true
  debug: false

browser-less:
  enabled: true
  port: 8081
  #reversal: http://127.0.0.1:8081
  disabled-gpu: true
  headless: new

custom-llm:
  - prefix: custom
    proxied: true
    reversal: https://models.inference.ai.azure.com
  - prefix: grok
    proxied: true
    reversal: https://api.x.ai/v1

matcher:
  - match: I do not engage
    over: ":\n"
    notice:
    regex: |
      "(?i)I do not engage .+:\n":""
  - match: <thinking>
    over: </thinking>
    notice:
    think_reason: true
    regex: |
      "(?s)<thinking>(.+)</thinking>":"$1"
```

### server 服务配置

`port` 启动端口

`proxied` 本地代理

`password` 访问密码，也可通过全局变量`PASSWORD`配置。对 `coze`、`you` 等这些cookie配置化的ai有效，对需要传 `authorization` 的ai无效

`think_reason` deepseek模型响应兼容reasoning_content字段

### browser-less 浏览器自动化配置

`enabled` 是否开启

`port` 数据请求端口

`reversal` 浏览器自动化服务分离时的访问地址，与`enabled`二选一

`disabled-gpu` 关闭gpu加速

`headless` 无头模式：true / false / new

### custom-llm 自定义v1桥接

`prefix` 模型前缀 - `${prefix}/gpt4o`

`reversal` llm目标地址

### matcher 响应token过滤器

`match` 字符块起始匹配

`over` 字符块结束匹配

`notice` 字符块起始匹配成功会响应制定字符串给前端，为空则忽略

`think_reason` 开启思考格式

`regex` 匹配成功后的内容正则处理： "regex": "$1"
