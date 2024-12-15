## 配置说明

```config.yaml
lmsys:
  token: '[ 95, 138 ]'
  model:
    - 'xxx'
```

*TIPS: 这两个值为fn_index、trigger_id 获取：进入[主页](https://lmarena.ai)，选择Direct Chat 发送一次对话，F12抓取join里的对应参数*

## 模型列表

```json
[
    "lmsys/chatgpt-4o-latest-20241120",
    "lmsys/gemini-exp-1121",
    "lmsys/gemini-exp-1114",
    "lmsys/chatgpt-4o-latest-20240903",
    "lmsys/gpt-4o-mini-2024-07-18",
    "lmsys/gpt-4o-2024-08-06",
    "lmsys/gpt-4o-2024-05-13",
    "lmsys/claude-3-5-sonnet-20241022",
    "lmsys/claude-3-5-sonnet-20240620",
    "lmsys/grok-2-2024-08-13",
    "lmsys/grok-2-mini-2024-08-13",
    "lmsys/gemini-1.5-pro-002",
    "lmsys/gemini-1.5-flash-002",
    "lmsys/gemini-1.5-flash-8b-001",
    "lmsys/gemini-1.5-pro-001",
    "lmsys/gemini-1.5-flash-001",
    "lmsys/llama-3.1-nemotron-70b-instruct",
    "lmsys/llama-3.1-nemotron-51b-instruct",
    "lmsys/llama-3.2-vision-90b-instruct",
    "lmsys/llama-3.2-vision-11b-instruct",
    "lmsys/llama-3.1-405b-instruct-bf16",
    "lmsys/llama-3.1-405b-instruct-fp8",
    "lmsys/llama-3.1-70b-instruct",
    "lmsys/llama-3.1-8b-instruct",
    "lmsys/llama-3.2-3b-instruct",
    "lmsys/llama-3.2-1b-instruct",
    "lmsys/hunyuan-standard-256k",
    "lmsys/mistral-large-2411",
    "lmsys/pixtral-large-2411",
    "lmsys/mistral-large-2407",
    "lmsys/yi-lightning",
    "lmsys/yi-vision",
    "lmsys/glm-4-plus",
    "lmsys/molmo-72b-0924",
    "lmsys/molmo-7b-d-0924",
    "lmsys/im-also-a-good-gpt2-chatbot",
    "lmsys/im-a-good-gpt2-chatbot",
    "lmsys/jamba-1.5-large",
    "lmsys/jamba-1.5-mini",
    "lmsys/gemma-2-27b-it",
    "lmsys/gemma-2-9b-it",
    "lmsys/gemma-2-2b-it",
    "lmsys/eureka-chatbot",
    "lmsys/claude-3-haiku-20240307",
    "lmsys/claude-3-sonnet-20240229",
    "lmsys/claude-3-opus-20240229",
    "lmsys/deepseek-v2.5",
    "lmsys/nemotron-4-340b",
    "lmsys/llama-3-70b-instruct",
    "lmsys/llama-3-8b-instruct",
    "lmsys/athene-v2-chat",
    "lmsys/qwen2.5-coder-32b-instruct",
    "lmsys/qwen2.5-72b-instruct",
    "lmsys/qwen-max-0919",
    "lmsys/qwen-plus-0828",
    "lmsys/qwen-vl-max-0809",
    "lmsys/gpt-3.5-turbo-0125",
    "lmsys/phi-3-mini-4k-instruct-june-2024",
    "lmsys/reka-core-20240904",
    "lmsys/reka-flash-20240904",
    "lmsys/c4ai-aya-expanse-32b",
    "lmsys/command-r-plus-08-2024",
    "lmsys/command-r-08-2024",
    "lmsys/codestral-2405",
    "lmsys/mixtral-8x22b-instruct-v0.1",
    "lmsys/f1-mini-preview",
    "lmsys/mixtral-8x7b-instruct-v0.1",
    "lmsys/pixtral-12b-2409",
    "lmsys/ministral-8b-2410",
    "lmsys/internvl2-26b",
    "lmsys/qwen2-vl-7b-instruct",
    "lmsys/internvl2-4b"
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
  "model": "lmsys/claude-3-5-sonnet-20241022",
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
{
    "temperature": [0.0~1.0],
    "max_tokens": [0.0~1.0],
    "top_p": [0.0~1.0]
}
```
