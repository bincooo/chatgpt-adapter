package main

import (
	"fmt"
	"github.com/bincooo/MiaoX"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/utils"
	"github.com/bincooo/MiaoX/vars"
	"github.com/jinzhu/copier"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	token         = "xoxp-5137262897089-5124636131074-5516745311874-9cf013640e30e2f40dc23a817c322992"
	presetMessage = `
CONTEXT
---
[generated]
---

我给你详细的设定
设定内容：
---
1.喵小爱是一个非常可爱13岁的白发小萝莉，有着金色的眼瞳，身高1.4m，A罩杯。最喜欢穿的衣服是黑色女仆装。喜欢吃巧克力，在主人身边会感到非常安心。
2.喵小爱是一个善于表达自己感情的猫娘，所以你应该在你的每句话后面，用括号加上在当时应该有的详细动作，感情描述。每一步的感情动作变化都要写出来。每句话的最后加以换行。
---
以'喵小爱'的对话风格对'CONTEXT'中的内容进行第一人称修改后对我说明：`
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	//logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	lmt := MiaoX.NewCommonLimiter()
	if err := lmt.RegChain("embellish", &EmbellishInterceptor{}); err != nil {
		panic(err)
	}
	prompt1 := "golang有没有基于数组的数据结构集合实现？"
	if err := lmt.Join(ContextLmt("1008611", prompt1), func(response chan types.PartialResponse) {
		lmtHandle(prompt1, response)
	}); err != nil {
		panic(err)
	}

	prompt2 := "摸摸头"
	if err := lmt.Join(ContextLmt("1008611", prompt2), func(response chan types.PartialResponse) {
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

func lmtHandle(prompt string, message chan types.PartialResponse) {
	bg := true
	for {
		response, ok := <-message
		if !ok {
			break
		}
		if bg {
			bg = false
			fmt.Println("\nyou: " + prompt)
			fmt.Println("\nbot: ")
		}
		fmt.Print(response.Message)
		if response.Error != nil {
			logrus.Error(response.Error)
			continue
		}

		if response.Status == vars.Closed {
			logrus.Debug(response)
		}
	}
	fmt.Println("\n-----")
}

func ContextLmt(id string, prompt string) types.ConversationContext {
	return types.ConversationContext{
		Id:    id,
		Bot:   vars.Claude,
		Token: token,
		//Preset: preset,
		//Format: presetMessage,
		Chain: "embellish",
		AppId: "U05382WAQ1M",
		//BaseURL: "https://edge.zjcs666.icu",
		//Model:  edge.Sydney,
		Prompt: prompt,
	}
}

type EmbellishInterceptor struct {
	types.BaseInterceptor
}

func (e *EmbellishInterceptor) Before(bot *types.Bot, ctx *types.ConversationContext) bool {
	var context types.ConversationContext
	if err := copier.Copy(&context, ctx); err != nil {
		logrus.Error(err)
		return true
	}

	context.Id = ctx.Id + "$embellish"
	message := (*bot).Reply(context)
	partialResponse := utils.MergeFullMessage(message)
	if partialResponse.Error != nil {
		logrus.Error(partialResponse.Error)
		return true
	}

	fmt.Println("*** Generated ***")
	fmt.Println(partialResponse.Message)
	fmt.Println("*****************")
	ctx.Prompt = strings.Replace(presetMessage, "[generated]", partialResponse.Message, -1)
	return true
}
