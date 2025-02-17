package cobra

import (
	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk"
	"github.com/iocgo/sdk/cobra"
	"github.com/iocgo/sdk/env"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type RootCommand struct {
	container *sdk.Container
	engine    *gin.Engine
	env       *env.Environment

	Port     int    `cobra:"port" short:"p" usage:"服务端口 port"`
	LogLevel string `cobra:"log" short:"L" usage:"日志级别: trace|debug|info|warn|error"`
	LogPath  string `cobra:"log-path" usage:"日志路径 log path"`
	Proxied  string `cobra:"proxies" short:"P" usage:"本地代理 proxies"`
	MView    bool   `cobra:"models" short:"M" usage:"展示模型列表"`
}

// @Cobra(name="cobra"
//
//	version = "v3.0.0-beta"
//	use     = "ChatGPT-Adapter"
//	short   = "GPT接口适配器"
//	long    = "GPT接口适配器。统一适配接口规范，集成了bing、claude-2，gemini...\n项目地址: https://github.com/bincooo/chatgpt-adapter"
//	run     = "Run"
//
// )
func New(container *sdk.Container, engine *gin.Engine, config string) (rc cobra.ICobra, err error) {
	environment, err := sdk.InvokeBean[*env.Environment](container, "")
	if err != nil {
		return
	}

	rc = cobra.ICobraWrapper(&RootCommand{
		container: container,
		engine:    engine,
		env:       environment,

		Port:     8080,
		LogLevel: "info",
		LogPath:  "log",
	}, config)
	return
}

func (rc *RootCommand) Run(cmd *cobra.Command, args []string) {
	if rc.env.GetBool("server.debug") {
		println(rc.container.HealthLogger())
	}

	if rc.MView {
		println("模型可用列表:")
		slice := sdk.ListInvokeAs[inter.Adapter](rc.container)
		for _, i := range slice {
			for _, mod := range i.Models() {
				println("- " + mod.Id)
			}
		}
		return
	}

	// init
	logger.InitLogger(
		rc.LogPath,
		LogLevel(rc.LogLevel),
	)
	Initialized(rc)
	inited.Initialized(rc.env)

	// gin
	addr := ":" + rc.env.GetString("server.port")
	println("Listening and serving HTTP on 0.0.0.0" + addr)
	if err := rc.engine.Run(addr); err != nil {
		panic(err)
	}
}

func Initialized(rc *RootCommand) {
	if rc.env.GetInt("server.port") == 0 {
		rc.env.Set("server.port", rc.Port)
	}
	if rc.Proxied != "" {
		rc.env.Set("server.proxied", rc.Proxied)
	}

	if rc.env.GetString("server.password") == "" {
		for _, item := range os.Environ() {
			if len(item) > 9 && item[:9] == "PASSWORD=" {
				rc.env.Set("server.password", item[9:])
				break
			}
		}
	}

	initFile(rc.env)
}

func LogLevel(lv string) logrus.Level {
	switch lv {
	case "trace":
		return logrus.TraceLevel
	case "debug":
		return logrus.DebugLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

func initFile(env *env.Environment) {
	_, err := os.Stat("config.yaml")
	if !os.IsNotExist(err) {
		return
	}

	content := "browser-less:\n  enabled: {enabled}\n  port: {port}\n  disabled-gpu: {gpu}\n  headless: {headless}\n  reversal: ${reversal}"
	content = strings.Replace(content, "{enabled}", env.GetString("browser-less.enabled"), 1)
	content = strings.Replace(content, "{port}", env.GetString("browser-less.port"), 1)
	content = strings.Replace(content, "{gpu}", env.GetString("browser-less.disabled-gpu"), 1)
	content = strings.Replace(content, "{headless}", env.GetString("browser-less.headless"), 1)
	content = strings.Replace(content, "{reversal}", env.GetString("browser-less.reversal"), 1)
	err = os.WriteFile("config.yaml", []byte(content), 0644)
	if err != nil {
		logger.Fatal(err)
	}
}
