### 喵小爱 - ai适配器

* 该库集成了openai-api、openai-web、claude for slack、bing 多款AI的对接接口
* 集成预设处理器，可对预设模版预处理

略...

#### 效果图

<details>
<summary>ZeroBot for QQ </summary>
<br/>
<a href="https://github.com/bincooo/ZeroBot-Plugin">【魔改ZeroBot-Plugin项目地址】</a>
<img src="resources/%E6%88%AA%E5%B1%8F2023-07-08%2000.02.13.png"  />
</details>

<details>
<summary>Terminal Example</summary>
<br/>
<img src="resources/%E6%88%AA%E5%B1%8F2023-07-08%2000.20.51.png"  />
</details>

<details>
<summary>类似LangChain的效果</summary>
<br/>
基础预设：
<img src="resources/%E6%88%AA%E5%B1%8F2023-07-09%2005.58.49.png" />
<br/>
设置执行链：
<code>
<pre>
	lmt := MiaoX.NewCommonLimiter()
	if err := lmt.RegChain("embellish", &EmbellishInterceptor{}); err != nil {
		panic(err)
	}
</pre>
</code>
<img src="resources/%E6%88%AA%E5%B1%8F2023-07-09%2006.03.24.png" />
<br/>
效果图：
<img src="resources/%E6%88%AA%E5%B1%8F2023-07-09%2006.08.03.png" />
</details>

Tips:

1.Claude for slack 配置以及token获取方式 [点我](https://github.com/Anyc66666666/claude-api)

2.Openai-web token获取方式: [登陆](http://chat.openai.com/)openai,  [访问链接](https://chat.openai.com/api/auth/session) 获取

3.Openai-web token获取方式：[登陆](https://platform.openai.com/)openai, `API keys` 处获取

4.Bing token获取方式:  登陆bing(有墙)，获取cookies中的_U值

### 待办

> Terminal Cli TODO <br>
> 增加了酒馆接口cli

> Socket or http TODO


### 编译

平台：
    `window` 、`linux` 、`darwin` <br>
示例（macos）：
```bash
GOOS=darwin GOARCH=amd64 go build cmd/exec.go
```

运行：
```bash
./exec -h
./exec --port 8080 --proxy http://127.0.0.1:7890
```