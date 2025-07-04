package cmd

import (
	"adapter/module/fiber/model"
	"fmt"

	"adapter/cmd/wrap"
	"adapter/module/common"
	"adapter/module/env"
	"adapter/module/fiber"
	"adapter/module/logger"
	"github.com/spf13/cobra"
)

var (
	cobraArgs = &CobraArgs{
		Port:     7860,
		LogLevel: "info",
		LogPath:  "log",
	}

	cmd = &cobra.Command{
		Use:     "adapter",
		Version: "v3.0.1-beta",
		Short:   "GPT接口适配器",
		Long:    "GPT接口适配器。统一适配接口规范，集成了bing、claude，gemini...\n项目地址: https://github.com/bincooo/chatgpt-adapter",

		Run: func(cmd *cobra.Command, args []string) {
			if cobraArgs.MView {
				println("模型可用列表:")
				var hasModel = false
				fiber.ModelEach(func(index int, model model.ModelEntity) {
					println("    - " + model.Id)
					hasModel = true
				})
				if !hasModel {
					println("    - 空 -")
				}
				return
			}

			// init
			logger.Initialized(
				cobraArgs.LogPath,
				LogLevel(cobraArgs.LogLevel),
			)

			environ, err := env.New()
			if err != nil {
				logger.Sugar().Fatalf("config.yaml is not exists; %v", err)
			}

			if port := env.Env.GetInt("server.port"); port > 0 {
				cobraArgs.Port = port
			}

			common.Initialized(environ)
			fiber.Initialized(fmt.Sprintf(":%d", cobraArgs.Port))
		},
	}
)

type CobraArgs struct {
	Port     int    `cobra:"port" short:"p" usage:"服务端口 port"`
	LogLevel string `cobra:"log" short:"L" usage:"日志级别: debug|info|warn|error"`
	LogPath  string `cobra:"log-path" usage:"日志路径 log path"`
	Proxied  string `cobra:"proxies" short:"P" usage:"本地代理 proxies"`
	MView    bool   `cobra:"models" short:"M" usage:"展示模型列表"`
}

func Initialized() {
	wrap.BindTags(cmd, wrap.ValueOf(cobraArgs))
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func LogLevel(lv string) logger.Level {
	switch lv {
	case "debug":
		return logger.DebugLevel
	case "warn":
		return logger.WarnLevel
	case "error":
		return logger.ErrorLevel
	default:
		return logger.InfoLevel
	}
}
