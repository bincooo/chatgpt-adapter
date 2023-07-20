package main

import (
	"fmt"
	"github.com/bincooo/MiaoX"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	clTypes "github.com/bincooo/claude-api/types"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

const (
	token  = "sk-ant-xxx"
	preset = ``

	A = "A:"
	H = "H:"
	S = "System:"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	//logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	manager := MiaoX.NewBotManager()
	//prompt1 := "「现在就开始吧」扑向你,把你衣服脱光"
	context := ContextLmt("1008611")
	for {
		fmt.Println("\n\nUser：")
	label:
		var prompt string
		_, err := fmt.Scanln(&prompt)
		if err != nil || prompt == "" {
			goto label
		}

		context.Prompt = prompt
		fmt.Println("Bot：")
		handle(context, manager)
		time.Sleep(time.Second)
	}
}

func handle(context types.ConversationContext, manager types.BotManager) {
	manager.Reply(context, func(partialResponse types.PartialResponse) {
		if partialResponse.Message != "" {
			fmt.Print(partialResponse.Message)
		}

		if partialResponse.Error != nil {
			logrus.Error(partialResponse.Error)
		}

		if partialResponse.Status == vars.Closed {
			return
		}
	})
}

func ContextLmt(id string) types.ConversationContext {
	return types.ConversationContext{
		Id:     id,
		Bot:    vars.Claude,
		Token:  token,
		Preset: preset,
		//Format: presetMessage,
		Chain: "replace,cache,claude2s",
		//AppId: "U05382WAQ1M",
		//BaseURL: "https://edge.zjcs666.icu",
		Proxy: "http://127.0.0.1:7890",
		Model: vars.Model4WebClaude2S,
		H:     Handle,
	}
}

// 「现在就开始吧」扑向你,把你衣服脱光
func Handle(rChan any) func(*types.CacheBuffer) error {
	pos := 0
	begin := false
	partialResponse := rChan.(chan clTypes.PartialResponse)
	return func(self *types.CacheBuffer) error {
		response, ok := <-partialResponse
		if !ok {
			self.Closed = true
			return nil
		}

		if response.Error != nil {
			self.Closed = true
			return response.Error
		}

		text := response.Text
		str := []rune(text)
		curStr := string(str[pos:])
		if index := strings.Index(curStr, A); index > -1 {
			if !begin {
				begin = true
				self.Cache += curStr[index:]
			} else {
				self.Cache += curStr[:index]
				self.Closed = true
				return nil
			}
		} else if index := strings.Index(curStr, H); index > -1 {
			self.Cache += curStr[:index]
			self.Closed = true
			return nil
		} else {
			self.Cache += string(str[pos:])
		}
		pos = len(str)
		return nil
	}
}
