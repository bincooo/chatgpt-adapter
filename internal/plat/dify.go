package plat

import (
	"context"
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"github.com/bincooo/dify-sdk-go"
	"io"
)

type DifyBot struct {
	sessions map[string]*difySession
}

type difySession struct {
	client         *dify.Client
	conversationId string
}

func NewDifyBot() types.Bot {
	return &DifyBot{
		sessions: make(map[string]*difySession),
	}
}

func (bot *DifyBot) Reply(ctx types.ConversationContext) chan types.PartialResponse {
	var message = make(chan types.PartialResponse)
	go func() {
		defer close(message)
		session, ok := bot.sessions[ctx.Id]
		if !ok {
			client := dify.NewClient(ctx.BaseURL, ctx.Token)
			session = &difySession{client, ""}
			bot.sessions[ctx.Id] = session
		}

		timeout, cancel := context.WithTimeout(context.TODO(), Timeout)
		defer cancel()

		req := &dify.ChatMessageRequest{
			Query:          ctx.Prompt,
			User:           "user",
			ConversationID: session.conversationId,
		}

		var err error
		var response chan dify.ChatMessageStreamChannelResponse
		if response, err = session.client.Api().ChatMessagesStream(timeout, req); err != nil {
			message <- types.PartialResponse{Error: err}
			return
		}
		conversationId := bot.handle(ctx, response, message)
		if conversationId != "" {
			session.conversationId = conversationId
			bot.sessions[ctx.Id] = session
		}
	}()
	return message
}

func (bot *DifyBot) Remove(id string) bool {
	return true
}

func (bot *DifyBot) handle(ctx types.ConversationContext, partialResponse chan dify.ChatMessageStreamChannelResponse, message chan types.PartialResponse) string {
	var r types.CacheBuffer
	var convId string
	if ctx.H != nil {
		r = types.CacheBuffer{
			H: ctx.H(partialResponse),
		}
	} else {
		r = types.CacheBuffer{
			H: func(self *types.CacheBuffer) error {
				response, ok := <-partialResponse
				if !ok {
					self.Closed = true
					return nil
				}

				if response.Err != nil {
					if response.Err == io.EOF {
						self.Closed = true
						return nil
					}
					return response.Err
				}
				self.Cache += response.Answer
				if convId == "" {
					convId = response.ConversationID
				}
				return nil
			},
		}
	}

	for {
		response := r.Read()
		message <- response
		if response.Status == vars.Closed {
			break
		}
	}
	return convId
}
