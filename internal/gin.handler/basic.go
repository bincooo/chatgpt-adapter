package handler

import (
	"bytes"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"slices"
	"strconv"
	"strings"
)

func Bind(port int, version, proxies string) {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()

	route.Use(crosHandler)
	route.Use(panicHandler)
	route.Use(whiteIPHandler)
	route.Use(tokenHandler)
	route.Use(proxiesHandler(proxies))
	route.Use(func(ctx *gin.Context) {
		ctx.Set("port", port)
	})

	route.GET("/", welcome(version))
	route.POST("/encipher", encipher)
	route.POST("/v1/chat/completions", completions)
	route.POST("/v1/object/completions", completions)
	route.POST("/proxies/v1/chat/completions", completions)
	route.POST("v1/images/generations", generations)
	route.POST("v1/object/generations", generations)
	route.POST("proxies/v1/images/generations", generations)
	route.GET("/proxies/v1/models", models)
	route.GET("/v1/models", models)
	route.Static("/file/tmp/", "tmp")

	route.POST("/anthropic/v1/messages", messages)

	route.Any("/socket.io/*any", gin.WrapH(plugin.IO.ServeHandler(nil)))

	addr := ":" + strconv.Itoa(port)
	logger.Info(fmt.Sprintf("server start by http://0.0.0.0%s/v1", addr))
	if err := route.Run(addr); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func encipher(context *gin.Context) {
	key := context.GetHeader("x-key")
	if key == "" {
		response.Error(context, -1, "请提供 `x-key` 请求头")
		return
	}

	defer context.Request.Body.Close()
	db, err := io.ReadAll(context.Request.Body)
	if err != nil {
		logger.Error(err)
		response.Error(context, -1, err)
		return
	}

	content, err := encrypt(key, string(db))
	if err != nil {
		logger.Error(err)
		response.Error(context, -1, err)
		return
	}

	context.String(http.StatusOK, content)
}

func encrypt(key string, content string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	ctx := pad(content)

	db := make([]byte, aes.BlockSize+len(ctx))
	iv := db[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	iToB := func(n int) []byte {
		x := int32(n)
		buffer := bytes.NewBuffer([]byte{})
		_ = binary.Write(buffer, binary.BigEndian, x)
		return buffer.Bytes()
	}

	cipher.NewCBCEncrypter(block, iv).CryptBlocks(db[aes.BlockSize:], ctx)
	toBytes := iToB(len(content))
	db = append(db, toBytes...)
	return hex.EncodeToString(db), nil
}

func pad(content string) (in []byte) {
	in = []byte(content)
	contentL := len(content)

	if remain := contentL % 16; remain != 0 {
		contentL = contentL + 16 - remain
		contentL = contentL - len(content)
		for i := 0; i < contentL; i++ {
			in = append(in, 0)
		}
	}
	return
}

func whiteIPHandler(context *gin.Context) {
	// 作用不大
	slice := pkg.Config.GetStringSlice("white-addr")
	if len(slice) != 0 {
		addr := getIp(context)
		if slices.Contains(slice, addr) {
			context.Next()
		} else {
			logger.Errorf("IP address %s is not whitelisted", addr)
			context.String(http.StatusForbidden, "refused")
			context.Abort()
		}
	}
}

func getIp(context *gin.Context) (ip string) {
	ip = context.ClientIP()
	header := context.GetHeader("X-Ip-Token")
	if header == "" {
		return
	}

	slice := strings.Split(header, ".")
	if len(slice) != 3 {
		return
	}

	db, err := base64.RawURLEncoding.DecodeString(slice[1])
	if err != nil {
		logger.Error(err)
		return
	}

	var obj map[string]interface{}
	if err = json.Unmarshal(db, &obj); err != nil {
		logger.Error(err)
		return
	}

	value, ok := obj["ip"].(string)
	if !ok {
		return
	}

	if pkg.Config.GetString("x-addr") != ip {
		return
	}

	ip = value
	return
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
		logger.Error(err)
	} else {
		logger.Infof("\n------ START REQUEST %s ---------\n%s", uid, data)
	}

	//处理请求
	context.Next()

	// 结束处理
	logger.Infof("\n------ END REQUEST %s ---------", uid)
}

func panicHandler(ctx *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("response error: %v", r)
			response.Error(ctx, -1, fmt.Sprintf("%v", r))
		}
	}()

	//处理请求
	ctx.Next()
}

func welcome(version string) gin.HandlerFunc {
	return func(context *gin.Context) {
		w := context.Writer
		str := strings.ReplaceAll(html, "VERSION", version)
		str = strings.ReplaceAll(str, "HOST", context.Request.Host)
		_, _ = w.WriteString(str)
	}
}

func models(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"object": "list",
		"data":   GlobalExtension.Models(),
	})
}
