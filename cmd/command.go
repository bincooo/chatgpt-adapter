package cmd

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/spf13/cobra"
)

var (
	version = "v2.0.0"
	proxies string
	port    int

	Cmd = &cobra.Command{
		Use:   "ChatGPT-Adapter",
		Short: "GPT接口适配器",
		Long: "GPT接口适配器。统一适配接口规范，集成了bing、claude-2，gemini...\n" +
			"项目地址：https://github.com/bincooo/chatgpt-adapter",
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			handler.Bind(port, version, proxies)
		},
	}
)

func Init() {
	pkg.Init()
	Cmd.PersistentFlags().StringVar(&proxies, "proxies", "", "本地代理 proxies")
	Cmd.PersistentFlags().IntVar(&port, "port", 8080, "服务端口 port")
}

func Exec() {
	Init()
	_ = Cmd.Execute()
}
