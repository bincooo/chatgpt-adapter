package main

import (
	"net/http"

	"github.com/bincooo/ja3"
	xtls "github.com/refraction-networking/utls"
	"github.com/xllm-go/g"

	_ "bypass/llm/arena"
	_ "bypass/llm/nexos"
)

var (
	Sdk = g.Sdk()
)

func main() {
	Sdk.OnInitialized(func() {
		Env := Sdk.Env()
		ja3.NewTransport(
			ja3.WithClientHelloID(xtls.HelloChrome_133),
			ja3.WithOriginalTransport(http.DefaultTransport.(*http.Transport)),
			ja3.WithProxy(Env.GetString("server.proxied")),
		)
	})
	g.Execute()
}
