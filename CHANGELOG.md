#  (2025-02-06)


### Bug Fixes

* abnormal relative path truncation ([3e335e7](https://github.com/bincooo/chatgpt-adapter/commit/3e335e7a09fbf320844557a3897e42296d58922c))
* **blackbox:** 补全请求参数 ([6224f08](https://github.com/bincooo/chatgpt-adapter/commit/6224f083def56166777e10997c8f64dc075c8f53))
* **coze:** bot名称配置化不同步 ([e641892](https://github.com/bincooo/chatgpt-adapter/commit/e6418929b62c88809840671dedf1ff7ff7a89856))
* cursor claude role error ([6007292](https://github.com/bincooo/chatgpt-adapter/commit/600729220a2aae4efde6367012fc55b2aa1e1b22))
* **cursor:** 时间戳编码错误修复([#92](https://github.com/bincooo/chatgpt-adapter/issues/92)) & 改进生成逻辑 ([c458b75](https://github.com/bincooo/chatgpt-adapter/commit/c458b759f6385d893a8fa9fe00cbe1f9ea642529))
* **cursor:** 设备码已开启检测，临时处理unauthorized ([a5c02ce](https://github.com/bincooo/chatgpt-adapter/commit/a5c02ceb1b5b1427892cd2ca42d13740c47f9316))
* model validate ([82f0db8](https://github.com/bincooo/chatgpt-adapter/commit/82f0db82ca9d5b96bc6ce25053f95176e5a8b47b))
* **windsurf:** claude-3-5-sonnet response empty tokens([#80](https://github.com/bincooo/chatgpt-adapter/issues/80)) ([d56dd57](https://github.com/bincooo/chatgpt-adapter/commit/d56dd576b23ee3413272432888494751cf356eda))
* **windsurf:** messages 长度导致空回复([#80](https://github.com/bincooo/chatgpt-adapter/issues/80)) ([1770147](https://github.com/bincooo/chatgpt-adapter/commit/1770147b9a826626cf0381e3428ad479c30663db))
* **windsurf:** 删除缓存的g_token([#80](https://github.com/bincooo/chatgpt-adapter/issues/80)) ([f3dc051](https://github.com/bincooo/chatgpt-adapter/commit/f3dc051cc3952153c4eab3afae85b60d56ae3b69))
* **windsurf:** 规避系统提示词检测导致对话失败 status code 500: [unknown] invalid argument ([e76c132](https://github.com/bincooo/chatgpt-adapter/commit/e76c132e41a19842a85ed761a9c3a91bffe4112d))
* **windsurf:** 过滤content mime数据类型 ([dc01eb7](https://github.com/bincooo/chatgpt-adapter/commit/dc01eb7d825659fb323300b3e9e2d9518b7405d1))
* **you:** 参数更新导致仅使用gpt回答([#83](https://github.com/bincooo/chatgpt-adapter/issues/83)) ([bf00426](https://github.com/bincooo/chatgpt-adapter/commit/bf0042643c3e360346903ee29bc7434a340083e4))
* **you:** 文件上传参数对齐 cannot unmarshal object into Go value of type string ([#97](https://github.com/bincooo/chatgpt-adapter/issues/97)) ([ca897af](https://github.com/bincooo/chatgpt-adapter/commit/ca897af947ffa2b6bfebe2d72a2b96201cd152d6))
* **you:** 短文本不以文件形式发送([#83](https://github.com/bincooo/chatgpt-adapter/issues/83)) ([9fd4bcc](https://github.com/bincooo/chatgpt-adapter/commit/9fd4bcc0e1f71a76c7e9c7e8d41d25cce2a6fcfe))
* 上下文处理时content超出默认cache buffer长度导致分割异常([#83](https://github.com/bincooo/chatgpt-adapter/issues/83)) ([cc546b8](https://github.com/bincooo/chatgpt-adapter/commit/cc546b8c90d5644f926be4b18a8ceb41e11afde0))
* 上下文处理时content超出默认cache buffer长度导致分割异常2([#83](https://github.com/bincooo/chatgpt-adapter/issues/83)) ([31208d9](https://github.com/bincooo/chatgpt-adapter/commit/31208d9222df1172f6723779a79a5ca0ad4333bc))
* 计算上下文tokens([#85](https://github.com/bincooo/chatgpt-adapter/issues/85)) ([fbb942c](https://github.com/bincooo/chatgpt-adapter/commit/fbb942cb28334b373c304d7e5ae3229ab0c01ca7))


### Features

* add bing model ([8e98806](https://github.com/bincooo/chatgpt-adapter/commit/8e9880669783a2ea869c1a1141dd910f2d02ad3d))
* add blackbox api ([c394df4](https://github.com/bincooo/chatgpt-adapter/commit/c394df48c5aa457d71aa7f7a9863d4b99a660a4a))
* **bing:** add accessToken header ([d0f3c82](https://github.com/bincooo/chatgpt-adapter/commit/d0f3c82d88e87e9f82b0a5f176d29e90878f66aa))
* **bing:** AI绘图适配 ([dc7ce7b](https://github.com/bincooo/chatgpt-adapter/commit/dc7ce7b6cd06c71a1c82a971e0d1a97551328560))
* **bing:** 修改cookie管理方式以解决有效期短问题 ([32b5cb5](https://github.com/bincooo/chatgpt-adapter/commit/32b5cb587b7b0a239aed6c7563bc0dd66bedf3e1))
* **bing:** 添加刷新accessToken方法 ([34660d1](https://github.com/bincooo/chatgpt-adapter/commit/34660d1d76c88ce881f69c55b531b428b083c524))
* **bing:** 自动过盾([#79](https://github.com/bincooo/chatgpt-adapter/issues/79)) ([5a71b4e](https://github.com/bincooo/chatgpt-adapter/commit/5a71b4e4aac8e5938efb2dafbc82dde2292a93cc))
* **bing:** 识图 ([81ddfc7](https://github.com/bincooo/chatgpt-adapter/commit/81ddfc7d06e237a67307f8100dd9ef5a27b9dc7c))
* **cursor:** 修改设备码计算方式 ([ecd30c0](https://github.com/bincooo/chatgpt-adapter/commit/ecd30c0ca39d5f296a745159e003122963725f59))
* cursor的基础实现 ([a2d580d](https://github.com/bincooo/chatgpt-adapter/commit/a2d580dbf5610b07fbecb4e5cb69ee43ac7b3d62))
* **deepseek:** think标签与api保持一致 ([8ad423b](https://github.com/bincooo/chatgpt-adapter/commit/8ad423bce314f74f706f7ddd52e3f8be161cdacd))
* **deepseek:** 添加模型 ([505fb7f](https://github.com/bincooo/chatgpt-adapter/commit/505fb7fd73c5fa51b215daf9e344c365b53f1478))
* **deepseek:** 自动删除会话 ([777fe6f](https://github.com/bincooo/chatgpt-adapter/commit/777fe6fae3547692ee43d7302acedb1c97fd2eed))
* **hf:** 添加抠图tag & 代码修改 ([7e0a567](https://github.com/bincooo/chatgpt-adapter/commit/7e0a5672627824349e008f72541eacc41f0d23f5))
* **windsurf:** add api ([d7b15f8](https://github.com/bincooo/chatgpt-adapter/commit/d7b15f843833009a1b91138a534986c5fcdf3890))
* **windsurf:** 添加deepseek模型 ([0070819](https://github.com/bincooo/chatgpt-adapter/commit/0070819ae7a1da0dca7a3ad5adfb9a711e49f9c7))
* **windsurf:** 缓存token ([ccfdfe5](https://github.com/bincooo/chatgpt-adapter/commit/ccfdfe542288c05611b2eabd11505a94bc9fe845))
* 配置化模型列表 ([#78](https://github.com/bincooo/chatgpt-adapter/issues/78)) ([1c53957](https://github.com/bincooo/chatgpt-adapter/commit/1c5395751148956a3123eba6d06792c36f3899e3))



