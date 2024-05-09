package handler

import (
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
)

func Bind(port int, version, proxies string) {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()

	route.Use(crosHandler)
	route.Use(panicHandler)
	route.Use(tokenHandler)
	route.Use(proxiesHandler(proxies))
	route.Use(func(ctx *gin.Context) {
		ctx.Set("port", port)
	})

	route.GET("/", index(version))
	route.POST("/v1/chat/completions", completions)
	route.POST("/v1/object/completions", completions)
	route.POST("/proxies/v1/chat/completions", completions)
	route.POST("v1/images/generations", generations)
	route.POST("v1/object/generations", generations)
	route.POST("proxies/v1/images/generations", generations)
	route.GET("/proxies/v1/models", models)
	route.GET("/v1/models", models)
	route.Static("/file/tmp/", "tmp")

	addr := ":" + strconv.Itoa(port)
	logrus.Info(fmt.Sprintf("server start by http://0.0.0.0%s/v1", addr))
	if err := route.Run(addr); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func proxiesHandler(proxies string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if proxies != "" {
			ctx.Set("proxies", proxies)
		}
	}
}

func tokenHandler(ctx *gin.Context) {
	token := ctx.Request.Header.Get("X-Api-Key")
	if token == "" {
		token = strings.TrimPrefix(ctx.Request.Header.Get("Authorization"), "Bearer ")
	}

	if token != "" {
		ctx.Set("token", token)
	}
}

func crosHandler(context *gin.Context) {
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

	uid := uuid.NewString()
	// 请求打印
	data, err := httputil.DumpRequest(context.Request, false)
	if err != nil {
		logrus.Error(err)
	} else {
		fmt.Printf("\n\n\n\n------ Start request %s  ---------\n%s\n", uid, data)
	}

	//处理请求
	context.Next()

	// 结束处理
	fmt.Printf("------ End request %s  ---------\n", uid)
}

func panicHandler(ctx *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("response error: %v", r)
			middle.ResponseWithV(ctx, -1, fmt.Sprintf("%v", r))
		}
	}()

	//处理请求
	ctx.Next()
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
				"id":       "claude",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "claude-adapter",
			}, {
				"id":       "claude-3-haiku-20240307",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "claude-adapter",
			}, {
				"id":       "claude-3-sonnet-20240229",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "claude-adapter",
			}, {
				"id":       "claude-3-opus-20240229",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "claude-adapter",
			}, {
				"id":       "bing",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "bing-adapter",
			}, {
				"id":       "coze",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "coze-adapter",
			}, {
				"id":       "gemini-1.0",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "gemini-adapter",
			}, {
				"id":       "gemini-1.5",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "gemini-adapter",
			}, {
				"id":       "command",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "cohere-adapter",
			}, {
				"id":       "command-r",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "cohere-adapter",
			}, {
				"id":       "command-light",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "cohere-adapter",
			}, {
				"id":       "command-light-nightly",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "cohere-adapter",
			}, {
				"id":       "command-nightly",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "cohere-adapter",
			}, {
				"id":       "command-r-plus",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "cohere-adapter",
			}, {
				"id":       "lmsys/claude-3-haiku-20240307",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/claude-3-sonnet-20240229",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/claude-3-opus-20240229",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/claude-2.1",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/reka-core-20240501",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/qwen-max-0428",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/qwen1.5-110b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/llama-3-70b-instruct",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/llama-3-8b-instruct",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/gemini-1.5-pro-api-0409-preview",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/snowflake-arctic-instruct",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/phi-3-mini-128k-instruct",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/mixtral-8x22b-instruct-v0.1",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/gpt-4-turbo-2024-04-09",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/gpt-3.5-turbo-0125",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/reka-flash",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/reka-flash-online",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/command-r-plus",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/command-r",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/gemma-1.1-7b-it",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/gemma-1.1-2b-it",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/mixtral-8x7b-instruct-v0.1",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/mistral-large-2402",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/mistral-medium",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/qwen1.5-72b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/qwen1.5-32b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/qwen1.5-14b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/qwen1.5-7b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/qwen1.5-4b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/zephyr-orpo-141b-A35b-v0.1",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/dbrx-instruct",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/starling-lm-7b-beta",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/llama-2-70b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/llama-2-13b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/llama-2-7b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/olmo-7b-instruct",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/vicuna-13b",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/yi-34b-chat",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/codellama-70b-instruct",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			}, {
				"id":       "lmsys/openhermes-2.5-mistral-7b",
				"object":   "model",
				"created":  1686935002,
				"owned_by": "lmsys-adapter",
			},
		},
	})
}
