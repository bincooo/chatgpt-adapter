package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/MiaoX"
	"github.com/bincooo/MiaoX/internal/plat"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	clTypes "github.com/bincooo/claude-api/types"
	"github.com/bincooo/claude-api/util"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	manager = MiaoX.NewBotManager()
	proxy   string
	port    int
	gen     bool
	count   int

	globalToken string
	muLock      sync.Mutex
)

const (
	H    = "H:"
	A    = "A:"
	S    = "System:"
	HARM = "I apologize, but I will not provide any responses that violate Anthropic's Acceptable Use Policy or could promote harm."
)

type rj struct {
	Prompt        string   `json:"prompt"`
	Model         string   `json:"model"`
	MaxTokens     int      `json:"max_tokens_to_sample"`
	StopSequences []string `json:"stop_sequences"`
	Temperature   float32  `json:"temperature"`
	TopP          float32  `json:"top_p"`
	TopK          float32  `json:"top_k"`
	Stream        bool     `json:"stream"`
}

type schema struct {
	TrimP bool `json:"trimP"` // 去掉头部Human
	TrimS bool `json:"trimS"` // 去掉尾部Assistant
	BoH   bool `json:"boH"`   // 响应截断H
	BoS   bool `json:"boS"`   // 响应截断System
}

func main() {
	_ = godotenv.Load()
	globalToken = loadEnvVar("CACHE_KEY", "")
	Exec()
}

func loadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}

func Exec() {
	types.CacheWaitTimeout = 1500 * time.Millisecond
	types.CacheMessageL = 20
	plat.Timeout = 3 * time.Minute // 3分钟超时，怎么的也够了吧

	var rootCmd = &cobra.Command{
		Use:   "MiaoX",
		Short: "MiaoX控制台工具",
		Long:  "MiaoX是集成了多款AI接口的控制台工具",
		Run:   Run,
	}

	rootCmd.Flags().StringVarP(&proxy, "proxy", "P", "", "本地代理")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "服务端口")
	rootCmd.Flags().BoolVarP(&gen, "gen", "g", false, "生成sessionKey")
	rootCmd.Flags().IntVarP(&count, "count", "c", 1, "生成sessionKey数量")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Run(cmd *cobra.Command, args []string) {
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
		//c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Next()
	})

	route.POST("/v1/complete", complete)
	addr := ":" + strconv.Itoa(port)
	fmt.Println("Start by http://127.0.0.1" + addr + "/v1")
	if err := route.Run(addr); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func genSessionKeys() {
	for i := 0; i < count; i++ {
		token, err := util.Login(proxy)
		if err != nil {
			panic(err)
		}
		fmt.Println("sessionKey=" + token)
	}
}

func complete(ctx *gin.Context) {
	var r rj

	token := ctx.Request.Header.Get("X-Api-Key")
	if err := ctx.BindJSON(&r); err != nil {
		responseError(ctx, err, r.Stream)
		return
	}

	IsClose := false
	context, err := createConversationContext(token, &r, func() bool { return IsClose })
	if err != nil {
		responseError(ctx, err, r.Stream)
		return
	}
	partialResponse := manager.Reply(*context, func(response types.PartialResponse) {
		if r.Stream {
			if response.Status == vars.Begin {
				ctx.Status(200)
				ctx.Header("Content-Type", "text/event-stream; charset=utf-8")
				ctx.Writer.Flush()
				return
			}

			if response.Error != nil {
				responseError(ctx, response.Error, r.Stream)
				return
			}

			if len(response.Message) > 0 {
				select {
				case <-ctx.Request.Context().Done():
					IsClose = true
				default:
					if !writeString(ctx, response.Message) {
						IsClose = true
					}
				}
			}

			if response.Status == vars.Closed {
				writeDone(ctx)
			}
		} else {
			select {
			case <-ctx.Request.Context().Done():
				IsClose = true
			default:
			}
		}
	})

	if !r.Stream && !IsClose {
		if partialResponse.Error != nil {
			responseError(ctx, partialResponse.Error, r.Stream)
			return
		}

		ctx.JSON(200, gin.H{
			"completion": partialResponse.Message,
		})
	}

	// 检查大黄标
	if token == "auto" && context.Model == vars.Model4WebClaude2S {
		if strings.Contains(partialResponse.Message, HARM) {
			// manager.Remove(context.Id, context.Bot)
			globalToken = ""
			fmt.Println("检测到大黄标（harm），下次请求将刷新cookie !")
		}
	}
}

func Handle(IsC func() bool, boH bool, boS bool) func(rChan any) func(*types.CacheBuffer) error {
	return func(rChan any) func(*types.CacheBuffer) error {
		pos := 0
		begin := false
		beginIndex := -1
		partialResponse := rChan.(chan clTypes.PartialResponse)
		return func(self *types.CacheBuffer) error {
			response, ok := <-partialResponse
			if !ok {
				// 清理一下残留
				self.Cache = strings.TrimSuffix(self.Cache, A)
				self.Cache = strings.TrimSuffix(self.Cache, S)
				self.Closed = true
				return nil
			}

			if IsC() {
				self.Closed = true
				return nil
			}

			if response.Error != nil {
				self.Closed = true
				return response.Error
			}

			text := response.Text
			str := []rune(text)
			self.Cache += string(str[pos:])
			pos = len(str)

			mergeMessage := self.Complete + self.Cache
			// 遇到“A:” 或者积累200字就假定是正常输出
			if index := strings.Index(mergeMessage, A); index > -1 {
				if !begin {
					begin = true
					beginIndex = index
					fmt.Println("---------\n", "1 输出中...")
				}

			} else if !begin && len(mergeMessage) > 200 {
				begin = true
				beginIndex = len(mergeMessage)
				fmt.Println("---------\n", "2 输出中...")
			}

			if begin {
				// fmt.Println("message: ", mergeMessage)
				// 遇到“H:”就结束接收
				if index := strings.Index(mergeMessage, H); boH && index > -1 && index > beginIndex {
					fmt.Println("---------\n", "遇到H:终止响应")
					if idx := strings.Index(self.Cache, H); idx >= 0 {
						self.Cache = self.Cache[:idx]
					}
					self.Closed = true
					return nil
				}
				// 遇到“System:”就结束接收
				if index := strings.Index(mergeMessage, S); boS && index > -1 && index > beginIndex {
					fmt.Println("---------\n", "遇到System:终止响应")
					if idx := strings.Index(self.Cache, S); idx >= 0 {
						self.Cache = self.Cache[:idx]
					}
					self.Closed = true
					return nil
				}
			}
			return nil
		}
	}
}

func createConversationContext(token string, r *rj, IsC func() bool) (*types.ConversationContext, error) {
	var (
		bot   string
		model string
		appId string
	)
	switch r.Model {
	case "claude-2.0":
		bot = vars.Claude
		model = vars.Model4WebClaude2S
	case "claude-1.0", "claude-1.2", "claude-1.3":
		bot = vars.Claude
		split := strings.Split(token, ",")
		token = split[0]
		if len(split) > 1 {
			appId = split[1]
		} else {
			return nil, errors.New("请在请求头中提供app-id")
		}
	default:
		return nil, errors.New("未知/不支持的模型`" + r.Model + "`")
	}

	message, s, err := trimMessage(r.Prompt)
	if err != nil {
		return nil, err
	}
	fmt.Println("-----------------------请求报文-----------------\n", message, "\n--------------------END-------------------")
	fmt.Println("Schema: ", s)
	if token == "auto" && globalToken == "" {
		muLock.Lock()
		defer muLock.Unlock()
		if globalToken == "" {
			globalToken, err = util.Login(proxy)
			if err != nil {
				fmt.Println("生成token失败： ", err)
				return nil, err
			}
			fmt.Println("生成token： " + globalToken)
			cacheKey(globalToken)
		}
	}

	if token == "auto" && globalToken != "" {
		token = globalToken
	}

	return &types.ConversationContext{
		Id:     "claude2",
		Token:  token,
		Prompt: message,
		Bot:    bot,
		Model:  model,
		Proxy:  proxy,
		H:      Handle(IsC, s.BoH, s.BoS),
		AppId:  appId,
	}, nil
}

func trimMessage(prompt string) (string, schema, error) {
	result := prompt
	// ====  Schema匹配 =======
	compileRegex := regexp.MustCompile(`schema\s?\{[^}]*}`)
	s := schema{
		TrimS: true,
		TrimP: true,
		BoH:   true,
		BoS:   false,
	}

	matchSlice := compileRegex.FindStringSubmatch(prompt)
	if len(matchSlice) > 0 {
		str := matchSlice[0]
		result = strings.Replace(result, str, "", -1)
		if err := json.Unmarshal([]byte(strings.TrimSpace(str[6:])), &s); err != nil {
			return "", s, err
		}
	}
	// =========================

	// ==== I apologize,[^\n]+ 道歉匹配 ======
	compileRegex = regexp.MustCompile(`I apologize[^\n]+`)
	result = compileRegex.ReplaceAllString(result, "")
	// =========================

	if s.TrimS {
		result = strings.TrimSuffix(result, "\n\nAssistant: ")
	}
	if s.TrimP {
		result = strings.TrimPrefix(result, "\n\nHuman: ")
	}

	result = strings.ReplaceAll(result, "A: ", "\nAssistant: ")
	result = strings.ReplaceAll(result, "H: ", "\nHuman: ")
	return strings.TrimSpace(result), s, nil
}

func responseError(ctx *gin.Context, err error, isStream bool) {
	if isStream {
		marshal, e := json.Marshal(gin.H{
			"completion": "Error: " + err.Error(),
		})
		fmt.Println("Error: ", err)
		if e != nil {
			fmt.Println("Error: ", e)
			return
		}
		ctx.String(200, "data: %s\n\ndata: [DONE]", string(marshal))
	} else {
		ctx.JSON(200, gin.H{
			"completion": "Error: " + err.Error(),
		})
	}
}

func writeString(ctx *gin.Context, content string) bool {
	c := strings.ReplaceAll(strings.ReplaceAll(content, "\n", "\\n"), "\"", "\\\"")
	if _, err := ctx.Writer.Write([]byte("\n\ndata: {\"completion\": \"" + c + "\"}")); err != nil {
		fmt.Println("Error: ", err)
		return false
	} else {
		ctx.Writer.Flush()
		return true
	}
}

func writeDone(ctx *gin.Context) {
	if _, err := ctx.Writer.Write([]byte("\n\ndata: [DONE]")); err != nil {
		fmt.Println("Error: ", err)
	}
}

// 缓存Key
func cacheKey(key string) {
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
	compileRegex := regexp.MustCompile(`^CACHE_KEY\s*=[^\n]*`)
	matchSlice := compileRegex.FindStringSubmatch(tmp)
	if len(matchSlice) > 0 {
		str := matchSlice[0]
		tmp = strings.Replace(tmp, str, "CACHE_KEY=\""+key+"\"", -1)
	} else {
		delimiter := ""
		if len(tmp) > 0 && !strings.HasSuffix(tmp, "\n") {
			delimiter = "\n"
		}
		tmp += delimiter + "CACHE_KEY=\"" + key + "\""
	}
	err = os.WriteFile(".env", []byte(tmp), 0664)
	if err != nil {
		fmt.Println("Error: ", err)
	}
}
