# [](https://github.com/bincooo/chatgpt-adapter/compare/v2.1.0-dev-2...v) (2024-05-29)



# [2.1.0-dev-2](https://github.com/bincooo/chatgpt-adapter/compare/v2.1.0-dev-1...v2.1.0-dev-2) (2024-05-29)


### Bug Fixes

* coze绘图失败 ([#18](https://github.com/bincooo/chatgpt-adapter/issues/18)) ([dd80970](https://github.com/bincooo/chatgpt-adapter/commit/dd80970f3665b0cb154faab6962bc2d8be6259f3))
* ToolChoice数据类型解析异常 ([622f4f0](https://github.com/bincooo/chatgpt-adapter/commit/622f4f0dfcd27daa78771bf92134b18b3a2ebc25))
* 上下文在使用[@n](https://github.com/n)标签时丢失消息 ([b35024e](https://github.com/bincooo/chatgpt-adapter/commit/b35024e303e1c9d8266c2bd2bd21d6e5a3d82b51))


### Features

* coze新增支持vision ([#19](https://github.com/bincooo/chatgpt-adapter/issues/19)) ([06d9361](https://github.com/bincooo/chatgpt-adapter/commit/06d9361546d4b1d2bbd64352f839dc7ee82aa597))
* coze新增支持指定bot模型 ([f19e417](https://github.com/bincooo/chatgpt-adapter/commit/f19e4172c671c9377abd2902708b5519b5a1bc35))
* gemini可指定默认tool ([bff652e](https://github.com/bincooo/chatgpt-adapter/commit/bff652e1e7db336c3a2d3980d0c58ca37aa9a83d))
* gemini安全配置外置 ([2c74128](https://github.com/bincooo/chatgpt-adapter/commit/2c74128be302c5e9495c44a5890f3b02a70469c4))
* gemini模型适配图片对话 ([11f217f](https://github.com/bincooo/chatgpt-adapter/commit/11f217f962f72a35c5a008d401ead167c8c1b631))
* 新增bing-online、bing-vision ([#22](https://github.com/bincooo/chatgpt-adapter/issues/22)) ([af69fa0](https://github.com/bincooo/chatgpt-adapter/commit/af69fa0577d13bfdf2631176f7c92cc9850bee70))
* 添加trace日志级别配置; gemini安全配置外置 ([decfe76](https://github.com/bincooo/chatgpt-adapter/commit/decfe7672182b732c8a578a65df2df5504bcb0ad))
* 空包返回时给下游回馈一个error ([27b19b0](https://github.com/bincooo/chatgpt-adapter/commit/27b19b0e7d475623c373c634180f117a3783ccfd))



# [2.1.0-dev-1](https://github.com/bincooo/chatgpt-adapter/compare/v2.0.2...v2.1.0-dev-1) (2024-05-23)


### Bug Fixes

* google toolCall ([929895d](https://github.com/bincooo/chatgpt-adapter/commit/929895db9aacb98a997d76b47061c1acc590f064))
* lmsys ctx done ([9ab34a5](https://github.com/bincooo/chatgpt-adapter/commit/9ab34a53c5b2ae30b1515b39a280d37f4c0f6532))
* lmsys fn_index参数变化导致403 ([354c9cb](https://github.com/bincooo/chatgpt-adapter/commit/354c9cbe9f82f7684b773a73c2e21a7612d29e69))
* lmsys 工具判断 ([e5a8e67](https://github.com/bincooo/chatgpt-adapter/commit/e5a8e67d7c7c44992794711cd9b87c36b3c2c067))
* lobechat 工具调用兼容 ([e470005](https://github.com/bincooo/chatgpt-adapter/commit/e47000506c3c50381008c99fad5b7dfc31988439))
* matcher漏字符 ([a91f26a](https://github.com/bincooo/chatgpt-adapter/commit/a91f26ae5ec72466d1af7e705632667f9fd76887))
* tool类消息解析遗漏 ([5b0a842](https://github.com/bincooo/chatgpt-adapter/commit/5b0a8423a21b7aeede4402208242d868bea3f719))
* 字符串hash值计算逻辑错误；删除goole无用代码 ([ac5e2d7](https://github.com/bincooo/chatgpt-adapter/commit/ac5e2d7395db94872a9fcb354a1167212b26485b))
* 画图参数获取错误 ([6669f24](https://github.com/bincooo/chatgpt-adapter/commit/6669f247b18114c34c955ad74bedb1d01683b06d))


### Features

* 1.修改google模型名称，添加flash模型；2.添加tool增强标签，用于工具选择默认 ([a37e593](https://github.com/bincooo/chatgpt-adapter/commit/a37e593109f2fe5255bf11eb0278c6b35f832323))
* bing调整参数，使其在一定程度上toolCall可用 ([f694ab8](https://github.com/bincooo/chatgpt-adapter/commit/f694ab8d4a3f6885b91e402aa894e1e93d7fefdc))
* cohere chat模式实现原生toolCall调用 ([5c943cb](https://github.com/bincooo/chatgpt-adapter/commit/5c943cbccc41cf55246bab0bc2cce8c89d0504b4))
* freeGpt35 toolCall实现 ([ec44e2e](https://github.com/bincooo/chatgpt-adapter/commit/ec44e2e489eb3a5d74f45d239afd9ec1cf6a9971))
* toolCall选择添加name 匹配 ([7344f93](https://github.com/bincooo/chatgpt-adapter/commit/7344f933117a3b10043ae7b4fec3e1222333ae32))
* 修改ToolCallCancel逻辑 ([96d3d52](https://github.com/bincooo/chatgpt-adapter/commit/96d3d522a99b0704dd872ae145b3a25ccbf760ba))
* 完善日志配置 ([74cdb2c](https://github.com/bincooo/chatgpt-adapter/commit/74cdb2c19ad5e7ce9d85c2c6dd5048776107989e))
* 工具调用添加任务拆解功能 ([410abff](https://github.com/bincooo/chatgpt-adapter/commit/410abffc7aa5bc77cb96966fc1ffc1e269893053))
* 日志级别参数化 ([ae9a93c](https://github.com/bincooo/chatgpt-adapter/commit/ae9a93ce1b835929787f0a2ed917b8f0a288b89f))
* 添加缓存机制，减少toolCall执行llm次数 ([fd87c22](https://github.com/bincooo/chatgpt-adapter/commit/fd87c22a40fe4d2d3f6d2befd9c36cbd0c86dbca))
* 绘图工具新增google接口 ([123b500](https://github.com/bincooo/chatgpt-adapter/commit/123b50033a110eaef359704dd758312f3baec646))



## [2.0.2](https://github.com/bincooo/chatgpt-adapter/compare/v2.0.1...v2.0.2) (2024-05-15)


### Bug Fixes

* 绘图地址错误；删除失效的模型 ([56a560a](https://github.com/bincooo/chatgpt-adapter/commit/56a560a6335a849f521e5765eab4ecb8b947fc9f))


### Features

* bing上下文处理修改 ([35a9a48](https://github.com/bincooo/chatgpt-adapter/commit/35a9a48c794a2c68353a07626a6e2903c076c9a1))
* CancelMather 单独抽离作为通用匹配器 ([bf2f7d9](https://github.com/bincooo/chatgpt-adapter/commit/bf2f7d9964f4c71f957c2744d07c1eb7a3816b6b))
* lmsys伪造请求头 ([b5d17d2](https://github.com/bincooo/chatgpt-adapter/commit/b5d17d210f84ce405feefb5f134c9521055401f6))
* lmsys更新模型 ([15cd8ef](https://github.com/bincooo/chatgpt-adapter/commit/15cd8ef1f7d19849cd04ddf027ca9fd59eb8c29c))
* lmsys特殊符号终止输出 ([4f1fe07](https://github.com/bincooo/chatgpt-adapter/commit/4f1fe074c1cc47489d506ec5c52a005d0b88eba3))
* lmsys违反规则消息体转error ([3a71b94](https://github.com/bincooo/chatgpt-adapter/commit/3a71b9451a83a08d2e8af76f91c570f6e60fe253))
* 接入lmsys的接口 ([7a1bf3d](https://github.com/bincooo/chatgpt-adapter/commit/7a1bf3d032625c68a40a66db1d2e476b02ccca84))
* 添加free gpt35接口 ([0dd5030](https://github.com/bincooo/chatgpt-adapter/commit/0dd50306fe54b05769025beaa0206f9eb8815b31))



## [2.0.1](https://github.com/bincooo/chatgpt-adapter/compare/v2.0.0...v2.0.1) (2024-04-19)


### Bug Fixes

* gemini合并消息，相邻的消息如果相同则会异常 400 \n\ngemini合并消息，相邻的消息如果相同则会异常（400 Please ensure that multiturn requests ends with a user role or a function response.） ([b879e1a](https://github.com/bincooo/chatgpt-adapter/commit/b879e1aa946f54e5428e2340b569081be76bb0b3))
* histories增强标记内容为空时 index out of bounds panic ([100cc55](https://github.com/bincooo/chatgpt-adapter/commit/100cc556bb7f84e504b503b0368c9b478ae39d8e))
* histories逻辑判断错误 ([4abeb59](https://github.com/bincooo/chatgpt-adapter/commit/4abeb59d4000de414a9b901c41b27caf54153d21))


### Features

* gemini修改流数据处理 ([c1aab4e](https://github.com/bincooo/chatgpt-adapter/commit/c1aab4eedafff1d1ab2a91a2d6d0aa507b28b17c))
* 接入gemini1.5官方api ([106a095](https://github.com/bincooo/chatgpt-adapter/commit/106a095b007349105577c164f3605c12520408d3))
* 添加usage属性以适配下游依赖该属性的请求 ([8241fb3](https://github.com/bincooo/chatgpt-adapter/commit/8241fb3304142862c7f34735f097782d2ac5a275))
* 添加图片放大api ([5da629f](https://github.com/bincooo/chatgpt-adapter/commit/5da629fd697d724516f0370515ddd50f01f1a408))



# [2.0.0](https://github.com/bincooo/chatgpt-adapter/compare/61bdd8f3dc041881923819b2750c2f02a8747afd...v2.0.0) (2024-04-09)


### Bug Fixes

* claude2对话请求改版 ([d5e1d31](https://github.com/bincooo/chatgpt-adapter/commit/d5e1d311bb9493c4aa97595c2b591c9bf465341e))
* coze bot版本信息修改(Some error occurred. Please try again ...) ([f791ad4](https://github.com/bincooo/chatgpt-adapter/commit/f791ad4d098e3e2e069e86ef4ab9d279ef9fc0c9))
* g1.5 解析残留 ([f02dcc3](https://github.com/bincooo/chatgpt-adapter/commit/f02dcc370a2a92e78a1bed93f3313b0b257448c0))
* golang for range 引用复制导致指向错误 ([6d3f502](https://github.com/bincooo/chatgpt-adapter/commit/6d3f502b0f8dbd440e233086270b606888bae2e0))
* histories增强标记无效 ([655f31f](https://github.com/bincooo/chatgpt-adapter/commit/655f31fdc110a5c5e84fa1a8fcae5329113827c9))
* tool_calls响应数据不完整 ([d46c2be](https://github.com/bincooo/chatgpt-adapter/commit/d46c2beb285a76be55561d80d40add1130a65099))
* 图片上传云接口不好使，删除这块功能 ([96e76a0](https://github.com/bincooo/chatgpt-adapter/commit/96e76a02283b9141ece54f8581e2388287d85cb8))


### Features

* (bing) function特殊处理 ([b1a309b](https://github.com/bincooo/chatgpt-adapter/commit/b1a309bf65c8706a04cdf6a7f18f578fd118f0a3))
* (bing) 合并历史对话，用文件的方式传输 ([c42de0b](https://github.com/bincooo/chatgpt-adapter/commit/c42de0b7bb97d5b0a8b219adf6c2934641811d42))
* (claude) 添加claude的实现 ([2b645c4](https://github.com/bincooo/chatgpt-adapter/commit/2b645c4f277fd7ca7adb73b72b676f27bac929c3))
* add coze api ([5632f45](https://github.com/bincooo/chatgpt-adapter/commit/5632f45c0c03deb7eff9dee02ffa705ca836b2a8))
* add gemini-1.5 ([98d68ae](https://github.com/bincooo/chatgpt-adapter/commit/98d68ae1c65a1c441d36afa2c886ad4ca52f8d60))
* add gemini, functionCall效果一言难尽 T_T ([3366d1b](https://github.com/bincooo/chatgpt-adapter/commit/3366d1b20dbddf76bfb0314eeb8a049b4016759e))
* agent change ([93dd24d](https://github.com/bincooo/chatgpt-adapter/commit/93dd24d0e73898628d8f1c5b7830294fd89ecab0))
* bing历史记录优化 ([a602af5](https://github.com/bincooo/chatgpt-adapter/commit/a602af58a864ae905cd205a47d9205479ad471ca))
* coze-api 需传入完整cookie; goole15灰度测试 ([5f30b15](https://github.com/bincooo/chatgpt-adapter/commit/5f30b151c351f735234167ff038d720763018160))
* coze删除cookie里的msToken ([14e8d4a](https://github.com/bincooo/chatgpt-adapter/commit/14e8d4a569e122a5b66e6bae4d9df5f442f36b96))
* coze通过计算对话token的长度，动态切换8k和128k；以达到延长对话次数 ([1665cdf](https://github.com/bincooo/chatgpt-adapter/commit/1665cdf118f06425cdd72c42d650f5e53671050d))
* gogle15提供参数可配 ([6af9111](https://github.com/bincooo/chatgpt-adapter/commit/6af9111eaf5dcc5a26ccd2fe0ffd456f1e54e27f))
* newbing请求实现 ([598c2a2](https://github.com/bincooo/chatgpt-adapter/commit/598c2a2ed80712b245d671840914ac1706d0230b))
* toolCall功能重写 ([5fe0c67](https://github.com/bincooo/chatgpt-adapter/commit/5fe0c677557f660e500cbc49517f74924c2e598c))
* 修改错误响应码，适配oneapi重试机制 ([674c59d](https://github.com/bincooo/chatgpt-adapter/commit/674c59dcc3eec968d31be8f6315f2219cdf4cf45))
* 加入免费的sd ([db2543a](https://github.com/bincooo/chatgpt-adapter/commit/db2543a33d1dcddc02899ff32a3d481d1ebef791))
* 将网络图片上传到Catbox再返回 ([84008b2](https://github.com/bincooo/chatgpt-adapter/commit/84008b2f01ad86279de75689e05671f01c21194e))
* 提示词修改 ([a429b68](https://github.com/bincooo/chatgpt-adapter/commit/a429b6828809af70d460afcf1225b65f4ba6056e))
* 支持设置 socks5 本地代理 ([dc88170](https://github.com/bincooo/chatgpt-adapter/commit/dc88170eb517c3d798065fe3565daa08a32c4b16))
* 新增cohere模型接口 ([295dcd9](https://github.com/bincooo/chatgpt-adapter/commit/295dcd9306ff2f239ea5f2fa683700830279d171))
* 流式响应符号匹配处理 ([6ba8e46](https://github.com/bincooo/chatgpt-adapter/commit/6ba8e46d7629b884fe0578157ac5f9dc9ba958fe))
* 添加Dockerfile部署文件 ([d5f6eca](https://github.com/bincooo/chatgpt-adapter/commit/d5f6eca8c099bda6228bd6ea1992f5d368de9aac))
* 添加histories flags处理（初始历史记录） ([4767d0e](https://github.com/bincooo/chatgpt-adapter/commit/4767d0efa4c5e8fcb69ebc645aa0933c4d5cb70f))
* 添加pg.dall-e-3绘画接口 ([2bfdc15](https://github.com/bincooo/chatgpt-adapter/commit/2bfdc15d226ad4376385531b92b3f9a85e799379))
* 添加日志 ([e435282](https://github.com/bincooo/chatgpt-adapter/commit/e4352821a9d644eab8f7d1919a0daebbe5c69a34))
* 添加日志 ([77f2b8a](https://github.com/bincooo/chatgpt-adapter/commit/77f2b8a71f6cb3cfd4fc455b2d7a7192c2f1691f))
* 添加特殊tags开关 ([b7e5e26](https://github.com/bincooo/chatgpt-adapter/commit/b7e5e26a9db0bea107fae52398e565888df10604))
* 添置特殊标签处理 ([9e12adc](https://github.com/bincooo/chatgpt-adapter/commit/9e12adccd5a10250ae0ec2b124e0e147ceee860c))
* 画图接口实现 ([40cc727](https://github.com/bincooo/chatgpt-adapter/commit/40cc727fe953b8fb2f46d1b950497ed10d2f3783))
* 自定义sd绘画 ([70b2671](https://github.com/bincooo/chatgpt-adapter/commit/70b2671346a9e2ab48b6fa3ad65ecc71c27a548d))
* 重构 ([61bdd8f](https://github.com/bincooo/chatgpt-adapter/commit/61bdd8f3dc041881923819b2750c2f02a8747afd))


### Reverts

* coze自动获取msToken会封号，还原以前的用法，可能会有历史对话问题 ([f1f4225](https://github.com/bincooo/chatgpt-adapter/commit/f1f4225f9e1d2fb1dee65e5153a2c4a834d28493))
* coze自动获取msToken会封号，还原以前的用法，可能会有历史对话问题 ([bc060e1](https://github.com/bincooo/chatgpt-adapter/commit/bc060e1ac3d825e631489ba6ad6e2013c6e36238))



