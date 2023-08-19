![](./images/describe-en.png)
<p align="center">
  <a href="README_en.md">English</a> |
  简体中文
</p>

### 喵小爱 - ai adaptor

* The library integrates `openai-api `, `openai-web`, `claude for slack`, `bing` and many AI docking interfaces
* Integrated preset processor for preprocessing preset templates

More...

#### Effect drawing

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
<summary>Similar to `LangChain` effect</summary>
<br/>
Base preset：
<img src="resources/%E6%88%AA%E5%B1%8F2023-07-09%2005.58.49.png" />
<br/>
Set Intercetper chain：
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
Effect：
<img src="resources/%E6%88%AA%E5%B1%8F2023-07-09%2006.08.03.png" />
</details>

Tips:

1.Claude for slack Configure and 'token' acquisition method [ClickMe](https://github.com/bincooo/claude-api)

2.Openai-web token Obtaining mode: [Login](http://chat.openai.com/)openai,  [ClickMe](https://chat.openai.com/api/auth/session) 获取

3.Openai-api token Obtaining mode：[Login](https://platform.openai.com/)openai, `API keys` 处获取

4.Bing token Obtaining mode:  Log in to `bing` and get the `_U` value in cookies

### TODO

> Terminal Cli TODO <br>
> Add SillyTavern cli

> Socket or http TODO


### BUILD

Platform：
    `windows` 、`linux` 、`darwin` <br>
Example（macos）：
```bash
GOOS=darwin GOARCH=amd64 go build cmd/exec.go
// arm64
GOARM=7 GOOS=linux GOARCH=arm64 go build cmd/exec.go
```
or
```bash
./build.sh
```

Run：
```bash
./exec -h
./exec --port 8080 --proxy http://127.0.0.1:7890
```


### SillyTavern



tips：


Supports streaming output and blocking output.

No need to install redundant dependencies.


SillyTavern Setting:<br/>
API: claude2.0<br/>
OpenAI Reverse Proxy: `http://127.0.0.1:8080/v1`<br/>
Proxy Password: `auto` or `seesionKey`<br/>



Proxy

1. window

   ```bash
   win-exec.exe --port 8080 --proxy http://127.0.0.1:7890
   ```

2. linux

   ```bash
   linux-exec --port 8080 --proxy http://127.0.0.1:7890
   ```

3. mac

   ```bash
   mac-exec --port 8080 --proxy http://127.0.0.1:7890
   ```

   

New: 

（2023-08-18）Scrap filled, turned on by default. Provides custom scrap text. Add a proxy self-test

```tex
// In the same directory `.env` file

# Fill text, default built-in random
PILE="Claude2.0 is so good."
# Fill in the maximum threshold, default 50000
PILE_SIZE=50000
```

（2023-08-16）Add scrap fill, turned on by default

Used：

Add the following code to your default body："pile", true (enable)，false (disable)

```tex
schema {
  "pile": true
}
```



（2023-08-16）The old mailbox is not available, Update the new mailbox.

```bash
Please select the mailbox suffix below:
        linshiyouxiang.net
        eur-rate.com
        deepyinc.com
        besttempmail.com
        5letterwordsfinder.com
        celebritydetailed.com
        comparisions.net
        randompickers.com
        bestwheelspinner.com
        justdefinition.com
```



（2023-08-12）Mailbox suffix built-in optional, try to compile arm64 `linux-exec-arm64`

```bash
Please select the mailbox suffix below:
        guerrillamail.biz
        guerrillamail.de
        guerrillamail.net
        guerrillamail.org
        guerrillamail.info
        guerrillamailblock.com
        pokemail.net
        spam4.me
        grr.la
```



（2023-08-09）fix: Fixed an issue with the change of stream read format on the official website.




（2023-07-27）Added automatic registration to get cookies (experimental), Rhubarb label automatic registration to refresh cookies

Used：

In the silly Tavern `Proxy Password` Fill out `auto`

When starting, you need to fill 'RECAPTCHA_KEY' and 'RECAPTCHA_TOKEN' in the '.env' file

`.env` Files and executable files are in the same directory：

```tex
[You folder]
		|-  win-exec.exe
		|-  .env
```

`RECAPTCHA_KEY`, `RECAPTCHA_TOKEN ` value in Claude. Ai landing page to fill in the email, click send after ` https://claude.ai/api/auth/send_code ` request in the developer tools

Whether there is an expiration time is unknown ？？？？

It is not bound to an IP address ？？？？

(Experimental function) Please test and study...





（2023-07-26）New Schema parameters (`trimP`, `trimS`, truncate `boH` when response meets H, System when response meets `boS`)

Used：

Add the following code to your default body：

```tex
schema {
  "trimS": false,
  "trimP": true,
  "boH": true,
  "boS": true
}
```

Example：
Main prompt
```tex
schema {"trimS": false, "trimP": true}
[preset...]
```
