package main

import (
	"fmt"
	"github.com/bincooo/MiaoX"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	token  = "auto"
	preset = ``
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	//logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	manager := AutoAI.NewBotManager()
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
		//Format: "【皮皮虾】: [content]",
		Chain: "replace,cache",
		//AppId: "U05382WAQ1M",
		//BaseURL: "https://edge.zjcs666.icu",
		Proxy: "http://127.0.0.1:7890",
		Model: vars.Model4WebClaude2S,
		//H:     Handle,
	}
}
