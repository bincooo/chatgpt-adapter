package main

import (
	"fmt"
	"github.com/bincooo/AutoAI/internal/plat"
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/claude-api/util"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
	"time"

	cmdtypes "github.com/bincooo/AutoAI/cmd/types"
	cmdutil "github.com/bincooo/AutoAI/cmd/util"
	cmdvars "github.com/bincooo/AutoAI/cmd/vars"
)

var (
	port  int
	gen   bool
	count int
)

const (
	VERSION = "v1.0.11"
)

func main() {
	_ = godotenv.Load()
	cmdvars.GlobalPile = cmdutil.LoadEnvVar("PILE", "")
	cmdvars.GlobalPileSize = cmdutil.LoadEnvInt("PILE_SIZE", 35000)
	cmdvars.GlobalToken = util.LoadEnvVar("CACHE_KEY", "")
	Exec()
}

func Exec() {
	types.CacheWaitTimeout = 0
	types.CacheMessageL = 14
	plat.Timeout = 5 * time.Minute // 5分钟超时，怎么的也够了吧

	var rootCmd = &cobra.Command{
		Use:     "MiaoX",
		Short:   "MiaoX控制台工具",
		Long:    "MiaoX是集成了多款AI接口的控制台工具\n  > 请在github star本项目获取最新版本: \nhttps://github.com/bincooo/MiaoX\nhttps://github.com/bincooo/claude-api",
		Run:     Run,
		Version: VERSION,
	}

	var esStr []string
	for _, bytes := range util.ES {
		esStr = append(esStr, string(bytes))
	}

	rootCmd.Flags().StringVarP(&cmdvars.Proxy, "proxy", "P", "", "本地代理 proxy network")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "服务端口 service port")
	rootCmd.Flags().BoolVarP(&gen, "gen", "g", false, "生成sessionKey")
	rootCmd.Flags().IntVarP(&count, "count", "c", 1, "生成sessionKey数量 generate count")
	rootCmd.Flags().StringVarP(&cmdvars.Bu, "base-url", "b", "", "第三方转发接口, 默认为官方 (Third party forwarding interface): https://claude.ai/api")
	rootCmd.Flags().StringVarP(&cmdvars.Suffix, "suffix", "s", "", "指定内置的邮箱后缀，如不指定随机选取 (Specifies the built-in mailbox suffix):\n\t"+strings.Join(esStr, "\n\t"))
	rootCmd.Flags().StringVarP(&cmdvars.I18nT, "i18n", "i", "zh", "国际化 (internationalization): zh, en")

	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func Run(*cobra.Command, []string) {
	switch cmdvars.I18nT {
	case "en":
	default:
		cmdvars.I18nT = "zh"
	}
	cmdvars.InitI18n()
	var esStr []string
	for _, bytes := range util.ES {
		esStr = append(esStr, string(bytes))
	}

	// 检查网络可用性
	if cmdvars.Proxy != "" {
		cmdutil.TestNetwork(cmdvars.Proxy)
	}

	if cmdvars.Suffix != "" && !cmdutil.Contains(esStr, cmdvars.Suffix) {
		logrus.Error(cmdvars.I18n("SUFFIX") + ":\n\t" + strings.Join(esStr, "\n\t"))
		os.Exit(1)
	}

	if gen {
		genSessionKeys()
		return
	}
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()

	route.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Next()
	})

	route.GET("/v1/models", models)
	route.POST("/v1/complete", complete)
	route.POST("/v1/chat/completions", completions)
	addr := ":" + strconv.Itoa(port)
	logrus.Info("Start by http://127.0.0.1" + addr + "/v1")
	if err := route.Run(addr); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func genSessionKeys() {
	for i := 0; i < count; i++ {
		email, token, err := util.LoginFor(cmdvars.Bu, cmdvars.Suffix, cmdvars.Proxy)
		if err != nil {
			logrus.Error("Error: ", email, err)
			os.Exit(1)
		}
		fmt.Println("email=" + email + "; sessionKey=" + token)
	}
}

func models(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"data": []gin.H{
			{"id": "claude-1.0"},
			{"id": "claude-2.0"},
			{"id": "BingAI"},
		},
	})
}

func complete(ctx *gin.Context) {
	var r cmdtypes.RequestDTO

	token := ctx.Request.Header.Get("X-Api-Key")
	if err := ctx.BindJSON(&r); err != nil {
		cmdutil.ResponseError(ctx, err.Error(), r.Stream, false)
		return
	}
	switch r.Model {
	case "claude-2.0", "claude-2":
	case "claude-1.0", "claude-1.2", "claude-1.3":
		cmdutil.DoClaudeComplete(ctx, token, &r)
	default:
		cmdutil.ResponseError(ctx, "未知的AI类型：`"+r.Model+"`", r.Stream, false)
	}
}

func completions(ctx *gin.Context) {
	var r cmdtypes.RequestDTO
	r.IsCompletions = true

	token := ctx.Request.Header.Get("X-Api-Key")
	if token == "" {
		token = strings.TrimPrefix(ctx.Request.Header.Get("Authorization"), "Bearer ")
	}
	if err := ctx.BindJSON(&r); err != nil {
		cmdutil.ResponseError(ctx, err.Error(), r.Stream, r.IsCompletions)
		return
	}
	switch r.Model {
	case "claude-2.0", "claude-2":
	case "claude-1.0", "claude-1.2", "claude-1.3":
		cmdutil.DoClaudeComplete(ctx, token, &r)
	case "BingAI":
		cmdutil.DoBingAIComplete(ctx, token, &r)
	default:
		cmdutil.ResponseError(ctx, "未知的AI类型：`"+r.Model+"`", r.Stream, true)
	}
}
