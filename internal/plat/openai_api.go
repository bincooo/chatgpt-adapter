package plat

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	wapi "github.com/bincooo/openai-wapi"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
)

type OpenAIAPIBot struct {
	token    string
	client   *openai.Client
	sessions map[string][]openai.ChatCompletionMessage
}

func (bot *OpenAIAPIBot) Reply(ctx types.ConversationContext) chan types.PartialResponse {
	var message = make(chan types.PartialResponse)
	go func() {
		defer close(message)
		stream, err := bot.makeCompletionStream(ctx)
		if err != nil {
			logrus.Error(err)
			return
		}
		defer stream.Close()

		r := CacheBuffer{
			H: func(self *CacheBuffer) error {
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					self.Closed = true
					return nil
				}

				if err != nil {
					logrus.Error(err)
					self.Closed = true
					return err
				}

				self.cache += response.Choices[0].Delta.Content
				return nil
			},
		}

		for {
			response := r.Read()
			message <- response
			if response.Status == vars.Closed {
				break
			}
		}

		messages := bot.sessions[ctx.Id]
		bot.sessions[ctx.Id] = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: r.complete,
		})
	}()
	return message
}

func (bot *OpenAIAPIBot) Reset(id string) bool {
	delete(bot.sessions, id)
	return true
}

func (bot *OpenAIAPIBot) makeCompletionStream(ctx types.ConversationContext) (stream *openai.ChatCompletionStream, err error) {
	model := ctx.Model
	if model == "" {
		model = openai.GPT3Dot5Turbo
	}
	request := openai.ChatCompletionRequest{
		Model:    model,
		Messages: bot.completionMessage(ctx),
		//MaxTokens:   ctx.MaxTokens,
		Temperature: 0.8,
		Stream:      true,
	}
	if bot.client == nil || bot.token != ctx.Token {
		bot.makeClient(ctx.Proxy, ctx.Token)
	}
	timeout, cancel := context.WithTimeout(context.TODO(), Timeout)
	defer cancel()
	return bot.client.CreateChatCompletionStream(timeout, request)
}

func (bot *OpenAIAPIBot) completionMessage(ctx types.ConversationContext) []openai.ChatCompletionMessage {
	messages, ok := bot.sessions[ctx.Id]
	if !ok {
		messages = make([]openai.ChatCompletionMessage, 0)
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: ctx.Prompt,
	})

	// 缓存30条记录
	if size := len(messages); size > 30 {
		messages = messages[size-30:]
	}
	bot.sessions[ctx.Id] = messages

	// 计算tokens
	var tokens = 0
	index := len(messages) - 1
	result := make([]openai.ChatCompletionMessage, 0)

	// 计算预设的tokens
	var preset []openai.ChatCompletionMessage
	if ctx.Preset != "" {
		if err := json.Unmarshal([]byte(ctx.Preset), &preset); err != nil {
			logrus.Error("预设解析失败")
		} else {
			for _, value := range preset {
				tokens += wapi.CalcTokens(value.Content)
			}
		}
	}

	for {
		if index < 0 {
			break
		}
		tokens += wapi.CalcTokens(messages[index].Content)
		// token溢出了
		if tokens > ctx.MaxTokens {
			// 把剩余的token也截取存入，限制最少长度为20
			if tokens > 20 {
				messages[index].Content = wapi.TokensEndSubstr(messages[index].Content, tokens)
				result = append(
					[]openai.ChatCompletionMessage{messages[index]},
					result...)
			}
			break
		}
		result = append(
			[]openai.ChatCompletionMessage{messages[index]},
			result...)
		index--
	}

	if len(preset) > 0 {
		result = append(preset, result...)
	}
	return result
}

func NewOpenAIAPIBot() types.Bot {
	return &OpenAIAPIBot{
		sessions: map[string][]openai.ChatCompletionMessage{},
	}
}

func (bot *OpenAIAPIBot) makeClient(proxy string, token string) {
	oc := openai.DefaultConfig(token)
	if proxy != "" {
		proxy, err := url.Parse(proxy)
		if err != nil {
			logrus.Error(err)
			os.Exit(0)
		}
		oc.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxy),
			},
		}
	}
	bot.token = token
	bot.client = openai.NewClientWithConfig(oc)
}