package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
)

func Bind(port int, version, proxies string) {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()
	route.Use(crosHandler())
	route.GET("/", index(version))
	route.POST("/v1/chat/completions", completions(proxies))
	route.POST("/v1/object/completions", completions(proxies))
	route.POST("/proxies/v1/chat/completions", completions(proxies))
	route.GET("/proxies/v1/models", models)
	route.GET("/v1/models", models)

	addr := ":" + strconv.Itoa(port)
	logrus.Info(fmt.Sprintf("server start by http://0.0.0.0%s/v1", addr))
	if err := route.Run(addr); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func crosHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method
		context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Origin", "*") // 设置允许访问所有域
		context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
		context.Header("Access-Control-Allow-Headers", "*")
		context.Header("Access-Control-Expose-Headers", "*")
		context.Header("Access-Control-Max-Age", "172800")
		context.Header("Access-Control-Allow-Credentials", "false")
		context.Set("content-type", "application/json")

		if method == "OPTIONS" {
			context.Status(http.StatusOK)
			return
		}
		//处理请求
		context.Next()
	}
}

func index(version string) gin.HandlerFunc {
	return func(context *gin.Context) {
		w := context.Writer
		format := "Start by http[s]://%s\n\nversion: %s\nproject: https://github.com/bincooo/chatgpt-adapter"
		if _, err := w.Write([]byte(fmt.Sprintf(format, context.Request.Host, version))); err != nil {
			logrus.Error(err)
		}
	}
}

func models(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"object": "list",
		"data": []gin.H{
			{
				"id":       "claude-2",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "claude-adapter",
			},
			{
				"id":       "bing",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "bing-adapter",
			},
			{
				"id":       "gemini",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "gemini-adapter",
			},
		},
	})
}
