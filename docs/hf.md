## 配置说明

```config.yaml
llm:
  model: bing
  token: 'xxx'
  reversal: 'http://127.0.0.1:8080/v1'

domain: http://127.0.0.1:8080

hf:
  dalle-4k:
    reversal: https://ijohn07-dalle-4k.hf.space
    fn: [3, 6]
    data: |
      [
        "{{prompt}}",
        "{{negative_prompt}}",
        true,
        "{{style}}",
        {{seed}},
        1024,
        1024,
        6,
        true
      ]
```

`llm`: openai格式的ai接口配置，用于生产tags

`domain`: 有些接口以图片文件方式返回，需配置domain来访问

`hf`: 内置接口实效时，可用该配置来定义新的hf空间参数



## 凭证列表

```json
[
    "sk-google-xl",
    "sk-dalle-4k",
    "sk-dalle-3-xl",
    "sk-animagine-xl-3.1"
]
```

## 请求示例

*TIPS: 该 authorization 为凭证列表中一个，代表不同的接口。*

```shell
curl -i -X POST \
   -H "Content-Type: application/json" \
   -H "Authorization: ${authorization}" \
   -d \
'{
  "model": "bing",
  "prompt": "<tag llm=false rmbg=true content="1girl, cat ear" /> pink hair",
  "size": "1024x1024",
  "style": "None",
  "quality": "Euler a"
}' \
 'http://127.0.0.1:8080/v1/images/generations'
```

#### `prompt`增强tag（可选）：

`<tag llm=true rmbg=true content="1girl, cat ear" />`

`llm`: 开启ai生成tag

`rmbg`: 开启删除背景

`content`: 不参与ai生成tag的提示tag
