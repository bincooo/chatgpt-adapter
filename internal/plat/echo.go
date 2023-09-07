package plat

import (
	"github.com/bincooo/AutoAI/types"
	"strconv"
	"time"
)

type EchoBot struct {
}

func (EchoBot) Reply(ctx types.ConversationContext) chan types.PartialResponse {
	var message = make(chan types.PartialResponse)
	go func() {
		defer close(message)
		for i := 0; i < 3; i++ {
			message <- types.PartialResponse{
				Message: ctx.Prompt + ", " + strconv.Itoa(i),
			}
			time.Sleep(3 * time.Second)
		}
	}()
	return message
}

func (EchoBot) Remove(id string) bool {
	return true
}

func NewEchoBot() types.Bot {
	return EchoBot{}
}
