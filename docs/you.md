## 配置说明

```config.yaml
you:
  custom: true
  cookies:
    - '_ga_2N7ZM9C56V=GS1.1.1732301239.1.1.1732301254.0.0.1111759234; you_subscription=freemium; ab.storage.userId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3AU2pDWigjrIDXMZPvlmrSOdSQS0yG%7Ce%3Aundefined%7Cc%3A1732301254524%7Cl%3A1732301254526; DSR=eyJhbGciOiJSUzI1NiIsImtpZCI6IlNLMmpJbnU3SWpjMkp1eFJad1psWHBZRUpQQkFvIiwidHlwIjoiSldUIn0.eyJhbXIiOlsiZW1haWwiXSwiYXV0aDBJZCI6bnVsbCwiZHJuIjoiRFNSIiwiZW1haWwiOiJwZm51dW1ra0BzaGFya2xhc2Vycy5jb20iLCJleHAiOjE3NjM3NTA4NTQsImdpdmVuTmFtZSI6IiIsImlhdCI6MTczMjMwMTI1NCwiaXNzIjoiUDJqSW50dFJNdVhweVlaTWJWY3NjNEM5WjBSVCIsImxhc3ROYW1lIjoiIiwibmFtZSI6IiIsInBpY3R1cmUiOiIiLCJzdHl0Y2hJZCI6bnVsbCwic3ViIjoiVTJwRFdpZ2pySURYTVpQdmxtclNPZFNRUzB5RyIsInRlbmFudEludml0YXRpb24iOm51bGwsInRlbmFudEludml0ZXIiOm51bGwsInVzZXJJZCI6IlUycERXaWdqcklEWE1aUHZsbXJTT2RTUVMweUciLCJ2ZXJpZmllZEVtYWlsIjp0cnVlfQ.XycIIb-rNopCToOzWdEHmkOmN1_XI1Qcli_BSrpAjjvEteZUoWrnzRwh-lg6dny2uKcJfee_DeH5j084HjrusteolFy30YEkiDmHd70D3Dndn4-FmRcWym9bI99i0QnP7krthDK9e4S8h9fLk94ipsQHIy9LJnBPGs0ycE9eUZpwYthHPgE7lysKOtc_lgE6q-pDdn2ZbJMhQuoB-PTaNOfXt4AkDz1nm4Qec2P0CamNMp1ZPnIpHz7mcPUwBdPII0L0fJl2aUI6eVUkvYIB-3MY_ha6M22M0Ilxtp0ci1X6Z1JByC2FARbTFdGuJtkvesV25ohKZ757zwX7kYhQCg; _gtmeec=eyJjdCI6IjBhNDk5MmVhNDQyYjUzZTNkY2E4NjFkZWFjMDlhOGQ0OTg3MDA0YTg0ODMwNzliMTI4NjEwODBlYTRhYTFiNTIiLCJzdCI6IjE5OGRlYTM5MmFmYzIxZDA3MGY4YWJmYzdjMDZkMTEwNjBmMWIyNzhlNWQ2MjUwMWJkNTdmMTE1OWE3MmViYWMiLCJ6cCI6IjhmOTFkMmI2NTgzNWUyYjcyMzFiYmM2ODNkOTVlNzk3MmI4MWRiZjQ3Yjc5ZjljZDM2N2ZkOGIwMWVmZDA5ZWYiLCJjb3VudHJ5IjoiNzlhZGIyYTJmY2U1YzZiYTIxNWZlNWYyN2Y1MzJkNGU3ZWRiYWM0YjZhNWUwOWUxZWYzYTA4MDg0YTkwNDYyMSJ9; _clsk=16fr0b0%7C1732301240090%7C1%7C0%7Cq.clarity.ms%2Fcollect; AF_DEFAULT_MEASUREMENT_STATUS=true; _gcl_au=1.1.93543382.1732301240; FPAU=1.1.93543382.1732301240; daily_query_date=Fri%20Nov%2022%202024; _ga=GA1.1.1950293868.1732301240; ab.storage.deviceId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3A917c796c-a9d5-8b2b-6faa-e6aec5674206%7Ce%3Aundefined%7Cc%3A1732301239546%7Cl%3A1732301254526; _clck=10u8ms5%7C2%7Cfr3%7C0%7C1787; youpro_subscription=false; daily_query_count=0; ab.storage.sessionId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3A5ad53bdc-bdb9-676e-6b53-5f181d5d1597%7Ce%3A1732303054530%7Cc%3A1732301254525%7Cl%3A1732301254530; FPID=FPID2.2.IMMix8nfVFnmJzQQdw%2BMkXZ4VzY%2BDMo%2BcaN8ZTpZ7Pg%3D.1732301240; DS=eyJhbGciOiJSUzI1NiIsImtpZCI6IlNLMmpJbnU3SWpjMkp1eFJad1psWHBZRUpQQkFvIiwidHlwIjoiSldUIn0.eyJhbXIiOlsiZW1haWwiXSwiYXV0aDBJZCI6bnVsbCwiZHJuIjoiRFMiLCJlbWFpbCI6InBmbnV1bWtrQHNoYXJrbGFzZXJzLmNvbSIsImV4cCI6MTczNDcyMDQ1NCwiZ2l2ZW5OYW1lIjoiIiwiaWF0IjoxNzMyMzAxMjU0LCJpc3MiOiJQMmpJbnR0Uk11WHB5WVpNYlZjc2M0QzlaMFJUIiwibGFzdE5hbWUiOiIiLCJuYW1lIjoiIiwicGljdHVyZSI6IiIsInJleHAiOiIyMDI1LTExLTIxVDE4OjQ3OjM0WiIsInN0eXRjaElkIjpudWxsLCJzdWIiOiJVMnBEV2lnanJJRFhNWlB2bG1yU09kU1FTMHlHIiwidGVuYW50SW52aXRhdGlvbiI6bnVsbCwidGVuYW50SW52aXRlciI6bnVsbCwidXNlcklkIjoiVTJwRFdpZ2pySURYTVpQdmxtclNPZFNRUzB5RyIsInZlcmlmaWVkRW1haWwiOnRydWV9.X1Tz7tV1tbSre0MUn_QES9cj9bbZjow75OKAvpODpM9kxGwOpajW3tdZm7sCVAyJUHtsJN6QN4R2OJExnzBf8_keKoAvXeDZ3llXLNc966C3YqFwm4o-5XFSJI7FKZW6cWRAyZxnkL4B4QtNZ5YZKAXk5Ft_8w0xCFIybn4WBcUAc9xeNFG-ukMY0qF0SDlYPSG1Lgx5Q2pDO-4boFAzV6QXV4H3rDiAdRstw_ceOeMS-77GsvP2-ERVRTAFIqxZwgCf8oXfnQ1vRqJOj2Vtd6d6LNsWq67lmed3Molm5FQ2_pn-Q_kPSTrLwFrcDXxzniiUCwGGu2mI8or1duPGmw; AF_SYNC=1732301240466; youchat_smart_learn=true; youchat_personalization=true; ai_model=gpt_4o; FPLC=ysIGegVWcFdg2mWhRCQe5ztvlGU3RGpoTrV0xV7UMyi9%2FtJbqqnXVRSekj%2FwdvETj2zQNlZ7BysjI%2F8elp7Tz0F57SiNSSPMIcxQoIJKav6Zxp0vRrWvkd2%2BQo0GVA%3D%3D; ld_context=%7B%22kind%22%3A%22user%22%2C%22key%22%3A%22ba1ebcd6-b741-43b7-abef-4f223a037b89%22%2C%22email%22%3A%22UNKNOWN%22%2C%22country%22%3A%22US%22%2C%22userAgent%22%3A%22Mozilla%2F5.0%20(X11%3B%20Linux%20x86_64)%20AppleWebKit%2F537.31%20(KHTML%2C%20like%20Gecko)%20Chrome%2F125.0.0.0%20Safari%2F537.36%20Edg%2F125.0.0.0%22%2C%22secUserAgent%22%3A%22%5C%22Microsoft%20Edge%5C%22%3Bv%3D%5C%22131%5C%22%2C%20%5C%22Chromium%5C%22%3Bv%3D%5C%22131%5C%22%2C%20%5C%22Not_A%20Brand%5C%22%3Bv%3D%5C%2224%5C%22%22%7D; uuid_guest_backup=1c0d441b-f25e-48f3-994d-cf98347a5275; uuid_guest=1c0d441b-f25e-48f3-994d-cf98347a5275; total_query_count=0; safesearch_guest=Moderate; afUserId=236f85d2-32da-4667-96f4-3fac36968ba2-p; FPGSID=1.1732301240.1732301240.G-WYGVQX1R23.CqhJy8e5kTR0fNgk7GzAqQxxxx'
```

`custom` 启动自定义模型，不存在会自动创建

## 模型列表

```json
[
    "you/gpt_4",
    "you/gpt_4o",
    "you/gpt_4o_mini",
    "you/gpt_4_turbo",
    "you/openai_o1",
    "you/openai_o1_mini",
    "you/claude_2",
    "you/claude_3_haiku",
    "you/claude_3_sonnet",
    "you/claude_3_5_sonnet",
    "you/claude_3_opus",
    "you/gemini_pro",
    "you/gemini_1_5_pro",
    "you/gemini_1_5_flash"
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
  "model": "you/gpt_4",
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


