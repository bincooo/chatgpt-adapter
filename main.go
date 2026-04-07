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
		ja3.NewTransport(
			ja3.WithClientHelloID(xtls.HelloChrome_133),
			ja3.WithOriginalTransport(http.DefaultTransport.(*http.Transport)),
			ja3.WithProxy(proxied),
		)

		http.DefaultClient.Jar, _ = cookiejar.New(
			&cookiejar.Options{
				PublicSuffixList: publicsuffix.List,
			},
		)
	})
	g.Execute()
}
