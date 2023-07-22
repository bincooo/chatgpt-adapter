package main

import (
	"encoding/json"
	"fmt"
	"github.com/bincooo/MiaoX"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	clTypes "github.com/bincooo/claude-api/types"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	manager = MiaoX.NewBotManager()
	proxy   string
	port    int
)

const (
	H = "H:"
	A = "A:"
	S = "System:"
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

func main() {
	Exec()
}

func Exec() {
	types.CacheWaitTimeout = 1500 * time.Millisecond
	types.CacheMessageL = 20

	var rootCmd = &cobra.Command{
		Use:   "MiaoX",
		Short: "MiaoX控制台工具",
		Long:  "MiaoX是集成了多款AI接口的控制台工具",
		Run:   Run,
	}

	rootCmd.Flags().StringVarP(&proxy, "proxy", "P", "", "本地代理")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "服务端口")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Run(cmd *cobra.Command, args []string) {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()
	route.POST("/v1/complete", complete)
	addr := ":" + strconv.Itoa(port)
	fmt.Println("Start by http://127.0.0.1" + addr)
	if err := route.Run(addr); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func complete(ctx *gin.Context) {
	var r rj

	token := ctx.Request.Header.Get("X-Api-Key")
	if err := ctx.BindJSON(&r); err != nil {
		responseError(ctx, err)
		return
	}

	fmt.Println("-----------------------请求报文-----------------\n", r, "\n--------------------END-------------------")

	partialResponse := manager.Reply(createConversationContext(token, &r), func(response types.PartialResponse) {
		if r.Stream {
			if response.Error != nil {
				responseError(ctx, response.Error)
				return
			}

			if len(response.Message) > 0 {
				writeString(ctx, response.Message)
			}

			if response.Status == vars.Closed {
				writeDone(ctx)
			}
		}
	})

	if !r.Stream {
		if partialResponse.Error != nil {
			responseError(ctx, partialResponse.Error)
			return
		}
		ctx.JSON(200, gin.H{
			"completion": partialResponse.Message,
		})
	}
}

func Handle(rChan any) func(*types.CacheBuffer) error {
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
			}

		} else if !begin && len(mergeMessage) > 200 {
			begin = true
			beginIndex = pos
		}

		if begin {
			// 遇到“H:”就结束接收
			if index := strings.Index(mergeMessage, H); index > -1 && index > beginIndex {
				self.Cache = strings.TrimSuffix(self.Cache, H)
				self.Closed = true
				return nil
			} else if index = strings.Index(mergeMessage, S); index > -1 && index > beginIndex {
				// 遇到“System:”就结束接收
				self.Cache = strings.TrimSuffix(self.Cache, S)
				self.Closed = true
				return nil
			}
		}
		return nil
	}
}

func createConversationContext(token string, r *rj) types.ConversationContext {
	return types.ConversationContext{
		Id:     "claude2",
		Token:  token,
		Prompt: r.Prompt,
		Bot:    vars.Claude,
		Model:  vars.Model4WebClaude2S,
		Proxy:  proxy,
		H:      Handle,
	}
}

func responseError(ctx *gin.Context, err error) {
	marshal, err := json.Marshal(gin.H{
		"completion": "Error: " + err.Error(),
	})
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	ctx.String(200, "data: %s\n\ndata: [DONE]", string(marshal))
}

func writeString(ctx *gin.Context, content string) {
	if _, err := ctx.Writer.Write([]byte("\n\ndata: {\"completion\": \"" + strings.ReplaceAll(content, "\n", "\\n") + "\"}")); err != nil {
		fmt.Println("Error: ", err)
	} else {
		ctx.Writer.Flush()
	}
}

func writeDone(ctx *gin.Context) {
	if _, err := ctx.Writer.Write([]byte("\n\ndata: [DONE]")); err != nil {
		fmt.Println("Error: ", err)
	}
}
