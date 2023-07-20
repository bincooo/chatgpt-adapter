package main

import (
	"fmt"
	"github.com/bincooo/MiaoX"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

const (
	token  = "sk-ant-xxx"
	preset = ``
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	//logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	lmt := MiaoX.NewCommonLimiter()
	prompt1 := "「现在就开始吧」扑向你,把你衣服脱光"
	if err := lmt.Join(ContextLmt("1008611", prompt1), func(response types.PartialResponse) {
		lmtHandle(prompt1, response)
	}); err != nil {
		panic(err)
	}

	prompt2 := "「还没有结束哦，今晚注定是不眠之夜呢」揉搓着你的双ru"
	if err := lmt.Join(ContextLmt("1008611", prompt2), func(response types.PartialResponse) {
		lmtHandle(prompt2, response)
	}); err != nil {
		panic(err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)
	<-shutdown

	lmt.Remove("1008611", vars.Claude)
	os.Exit(0)
}

func lmtHandle(prompt string, message types.PartialResponse) {
	if message.Status == vars.Begin {
		fmt.Println("\nyou: " + prompt)
		fmt.Println("\nbot: ")
	}

	fmt.Print(message.Message)
	if message.Error != nil {
		logrus.Error(message.Error)
	}

	if message.Status == vars.Closed {
		fmt.Println("\n-----")
	}
}

func ContextLmt(id string, prompt string) types.ConversationContext {
	return types.ConversationContext{
		Id:     id,
		Bot:    vars.Claude,
		Token:  token,
		Preset: preset,
		//Format: presetMessage,
		Chain: "replace,cache,claude2s",
		//AppId: "U05382WAQ1M",
		//BaseURL: "https://edge.zjcs666.icu",
		Proxy:  "http://127.0.0.1:7890",
		Model:  vars.Model4WebClaude2S,
		Prompt: prompt,
	}
}
