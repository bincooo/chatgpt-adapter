package util

import (
	"encoding/json"
	"fmt"
	cmdtypes "github.com/bincooo/AutoAI/cmd/types"
	"github.com/bincooo/AutoAI/cmd/vars"
	"github.com/bincooo/requests"
	"github.com/bincooo/requests/url"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	"strings"
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
		handle := func(val string) map[string]string {
			val = strings.TrimSpace(val)
			if strings.HasPrefix(val, "Assistant:") {
				return map[string]string{
					"role":    "assistant",
					"content": strings.TrimSpace(strings.TrimPrefix(val, "Assistant:")),
				}
			}
			if strings.HasPrefix(val, "System:") {
				return map[string]string{
					"role":    "system",
					"content": strings.TrimSpace(strings.TrimPrefix(val, "System:")),
				}
			}
			return map[string]string{
				"role":    "user",
				"content": strings.TrimSpace(strings.TrimPrefix(val, "Human:")),
			}
		}

		content := r.Messages[0]["content"]
		content = strings.ReplaceAll(content, "<histories></histories>", "")
		content = strings.TrimSuffix(content, "\nAssistant: ")

		content = strings.ReplaceAll(content, "<histories>", "<|[1]|><histories>")
		contents := strings.Split(content, "<|[1]|>")
		temp := contents
		contents = []string{}
		for _, human := range temp {
			if human == "" {
				continue
			}
			histories := strings.Split(human, "</histories>")
			contents = append(contents, histories...)
		}

		splitHandle := func(item string) []map[string]string {
			messages := make([]map[string]string, 0)
			item = strings.ReplaceAll(item, "\nHuman:", "<|[1]|>\nHuman:")
			humans := strings.Split(item, "<|[1]|>\n")
			temp = humans
			humans = []string{}
			for _, human := range temp {
				if human == "" {
					continue
				}
				human = strings.ReplaceAll(human, "\nAssistant:", "<|[1]|>\nAssistant:")
				assistants := strings.Split(human, "<|[1]|>\n")
				humans = append(humans, assistants...)
			}
			temp = humans
			humans = []string{}
			for _, human := range temp {
				if human == "" {
					continue
				}
				human = strings.ReplaceAll(human, "\nSystem:", "<|[1]|>\nSystem:")
				systems := strings.Split(human, "<|[1]|>\n")
				humans = append(humans, systems...)
			}
			for _, human := range humans {
				if human == "" {
					continue
				}
				messages = append(messages, handle(human))
			}
			return messages
		}

		messages := make([]map[string]string, 0)
		for _, item := range contents {
			if item == "Assistant: " || item == "Here is the chat histories between human and assistant, inside <histories></histories> XML tags." {
				continue
			}
			if strings.HasPrefix(item, "<histories>") {
				item = strings.TrimPrefix(item, "<histories>")
				item = strings.TrimSuffix(item, "</histories>")
				messages = append(messages, splitHandle(item)...)
				continue
			}
			messages = append(messages, splitHandle(item)...)
		}
		r.Messages = messages
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
	message = strings.ReplaceAll(message, "\n", Enter)
	message = strings.ReplaceAll(message, "\"", DQM)
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

// 缓存CACHE_KEY
func CacheKey(key, value string) {
	// 文件不存在...   就创建吧
	if _, err := os.Lstat(".env"); os.IsNotExist(err) {
		if _, e := os.Create(".env"); e != nil {
			fmt.Println("Error: ", e)
			return
		}
	}

	bytes, err := os.ReadFile(".env")
	if err != nil {
		fmt.Println("Error: ", err)
	}
	tmp := string(bytes)
	compileRegex := regexp.MustCompile(`(\n|^)` + key + `\s*=[^\n]*`)
	matchSlice := compileRegex.FindStringSubmatch(tmp)
	if len(matchSlice) > 0 {
		str := matchSlice[0]
		if strings.HasPrefix(str, "\n") {
			str = str[1:]
		}
		tmp = strings.Replace(tmp, str, key+"=\""+value+"\"", -1)
	} else {
		delimiter := ""
		if len(tmp) > 0 && !strings.HasSuffix(tmp, "\n") {
			delimiter = "\n"
		}
		tmp += delimiter + key + "=\"" + value + "\""
	}
	err = os.WriteFile(".env", []byte(tmp), 0664)
	if err != nil {
		fmt.Println("Error: ", err)
	}
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
