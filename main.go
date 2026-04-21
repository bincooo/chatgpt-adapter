package main

import (
	"net/http"
	"net/http/cookiejar"

	"github.com/bincooo/ja3"
	"github.com/xllm-go/g"
	"golang.org/x/net/publicsuffix"

	xtls "github.com/refraction-networking/utls"

	_ "bypass/llm/arena"
	_ "bypass/llm/chatgpt"
	_ "bypass/llm/nexos"

	_ "bypass/image/hf"
)

var (
	Sdk = g.Sdk()
)

func main() {
	Sdk.OnInitialized(func() {
		Env := Sdk.Env()
		proxied := Env.GetString("server.proxied")
		http.DefaultTransport = ja3.NewTransport(
			ja3.WithClientHelloID(xtls.HelloChrome_133),
			ja3.WithOriginalTransport(http.DefaultTransport.(*http.Transport)),
			ja3.WithProxy(proxied),
		)
		// idle 配置
		if transport, ok := http.DefaultTransport.(interface{ Rule(string, int) }); ok {
			transport.Rule("*animagine-xl-4-0.hf.space", 3)       // 空闲3s关闭, 用于代理连接池的切换
			transport.Rule("*z-image-turbo-lora-dlc.hf.space", 3) // 空闲3s关闭, 用于代理连接池的切换
		}

		http.DefaultClient.Jar, _ = cookiejar.New(
			&cookiejar.Options{
				PublicSuffixList: publicsuffix.List,
			},
		)
	}, 0)
	g.Execute()
}
