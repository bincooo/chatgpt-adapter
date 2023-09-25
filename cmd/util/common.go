package util

import (
	"encoding/json"
	"fmt"
	cmdtypes "github.com/bincooo/AutoAI/cmd/types"
	"github.com/bincooo/AutoAI/cmd/util/dify"
	"github.com/bincooo/AutoAI/cmd/vars"
	"github.com/bincooo/requests"
	"github.com/bincooo/requests/url"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	// 回车
	Enter = "\u000A"
	// 双引号
	DQM = "\u0022"
)

func CleanToken(token string) {
	if token == "auto" {
		vars.GlobalToken = ""
	}
}

// dify的LocalAI请求数据切割成适配的上下文
func prepare(ctx *gin.Context, r *cmdtypes.RequestDTO) {
	// isDify
	if ctx.Request.RequestURI == "/dify/v1/chat/completions" && len(r.Messages) > 0 {
		dify.ConvertMessages(r)
	}
}

func ResponseError(ctx *gin.Context, err string, isStream bool, isCompletions bool, wd bool) {
	logrus.Error(err)
	if isStream {
		marshal, e := json.Marshal(BuildCompletion(isCompletions, "Error: "+err))
		if e != nil {
			return
		}
		if wd {
			ctx.String(200, "data: %s\n\ndata: [DONE]", string(marshal))
		} else {
			ctx.String(200, "data: %s", string(marshal))
		}
	} else {
		ctx.JSON(200, BuildCompletion(isCompletions, "Error: "+err))
	}
}

func WriteString(ctx *gin.Context, content string, isCompletions bool) bool {
	completion := BuildCompletion(isCompletions, content)
	marshal, err := json.Marshal(completion)
	if err != nil {
		logrus.Error(err)
		return false
	}
	if _, err = ctx.Writer.Write([]byte("data: " + string(marshal) + "\n\n")); err != nil {
		logrus.Error(err)
		return false
	} else {
		ctx.Writer.Flush()
		return true
	}
}

func WriteDone(ctx *gin.Context, isCompletions bool) {
	// 结尾img标签会被吞？？多加几个换行试试
	var completion string
	if isCompletions {
		completion = "data: {\"choices\": [ { \"message\": {\"role\":\"assistant\", \"content\": \"" + Enter + Enter + "\"} } ]}\n\n"
	} else {
		completion = "data: {\"completion\": \"" + Enter + Enter + "\"}\n\n"
	}
	if _, err := ctx.Writer.Write([]byte(completion)); err != nil {
		logrus.Error(err)
	}
	if _, err := ctx.Writer.Write([]byte("data: [DONE]")); err != nil {
		logrus.Error(err)
	}
}

func BuildCompletion(isCompletions bool, message string) gin.H {
	var completion gin.H
	if isCompletions {
		content := gin.H{"content": message, "role": "assistant"}
		completion = gin.H{
			"choices": []gin.H{
				{
					"message": content,
					"delta":   content,
				},
			},
		}
	} else {
		completion = gin.H{
			"completion": message,
		}
	}
	return completion
}

// 判断切片是否包含子元素
func Contains[T comparable](slice []T, t T) bool {
	if len(slice) == 0 {
		return false
	}

	return ContainFor(slice, func(item T) bool {
		return item == t
	})
}

// 判断切片是否包含子元素， condition：自定义判断规则
func ContainFor[T comparable](slice []T, condition func(item T) bool) bool {
	if len(slice) == 0 {
		return false
	}

	for idx := 0; idx < len(slice); idx++ {
		if condition(slice[idx]) {
			return true
		}
	}
	return false
}

func TestNetwork(proxy string) {
	req := url.NewRequest()
	req.Timeout = 5 * time.Second
	req.Proxies = proxy
	req.AllowRedirects = false
	response, err := requests.Get("https://claude.ai/login", req)
	if err == nil && response.StatusCode == 200 {
		fmt.Println("🎉🎉🎉 Network success! 🎉🎉🎉")
		req = url.NewRequest()
		req.Timeout = 5 * time.Second
		req.Proxies = proxy
		req.Headers = url.NewHeaders()
		response, err = requests.Get("https://iphw.in0.cc/ip.php", req)
		if err == nil {
			compileRegex := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
			ip := compileRegex.FindStringSubmatch(response.Text)
			if len(ip) > 0 {
				country := ""
				response, err = requests.Get("https://opendata.baidu.com/api.php?query="+ip[0]+"&co=&resource_id=6006&oe=utf8", nil)
				if err == nil {
					obj, e := response.Json()
					if e == nil {
						if status, ok := obj["status"].(string); ok && status == "0" {
							country = obj["data"].([]interface{})[0].(map[string]interface{})["location"].(string)
						}
					}
				}

				fmt.Println(vars.I18n("IP") + ": " + ip[0] + ", " + country)
			}
		}
	} else {
		fmt.Println("🚫🚫🚫 " + vars.I18n("NETWORK_DISCONNECTED") + " 🚫🚫🚫")
	}
}

func LoadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}

func LoadEnvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		logrus.Error(err)
		os.Exit(-1)
	}
	return result
}