package main

import (
	"fmt"
	"github.com/bincooo/AutoAI"
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	token = "app-cU82sYQMQN35Kww9VjGKMHNE"
)

func init() {
	//logrus.SetLevel(logrus.DebugLevel)
	logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	manager := AutoAI.NewBotManager()
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
		Bot:     vars.Dify,
		Token:   token,
		Chain:   "replace",
		BaseURL: "http://127.0.0.1:81",
		//Model:   edge.Sydney,
	}
}
