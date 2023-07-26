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
	token         = "123"
	preset        = ``
	presetMessage = ``
)

func init() {
	//logrus.SetLevel(logrus.DebugLevel)
	logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	manager := MiaoX.NewBotManager()
	context := Context()
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
		time.Sleep(time.Second)
	}
}

func Context() types.ConversationContext {
	return types.ConversationContext{
		Id: "1008611",
		//Bot:     vars.Bing,
		Bot:       vars.OpenAIAPI,
		Token:     token,
		Preset:    preset,
		Format:    presetMessage,
		Chain:     "replace,cache,assist",
		BaseURL:   "http://127.0.0.1:8080/v1",
		Model:     "gpt-3.5-turbo",
		MaxTokens: 10000,
	}
}
