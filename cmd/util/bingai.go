package util

import (
	"errors"
	"fmt"
	cmdtypes "github.com/bincooo/AutoAI/cmd/types"
	cmdvars "github.com/bincooo/AutoAI/cmd/vars"
	"github.com/bincooo/AutoAI/store"
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"github.com/bincooo/edge-api"
	"github.com/bincooo/edge-api/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strings"
)

var (
	bingBaseURL = ""
	bingAIToken = ""
)

func init() {
	bingAIToken = LoadEnvVar("BING_TOKEN", "")
	bingBaseURL = LoadEnvVar("BING_BASE_URL", "")
}

func DoBingAIComplete(ctx *gin.Context, token string, r *cmdtypes.RequestDTO, wd bool) {
	IsClose := false
	if token == "" || token == "auto" {
		token = bingAIToken
	}
	fmt.Println("TOKEN_KEY: " + token)
	prepare(ctx, r)
	context, err := createBingAIConversation(r, token, func() bool { return IsClose })
	if err != nil {
		responseBingAIError(ctx, err, r.Stream, r.IsCompletions, token, wd)
		return
	}
	partialResponse := cmdvars.Manager.Reply(*context, func(response types.PartialResponse) {
		if r.Stream {
			if response.Error != nil {
				responseBingAIError(ctx, response.Error, r.Stream, r.IsCompletions, token, wd)
				return
			}

			if response.Status == vars.Begin {
				ctx.Status(200)
				ctx.Header("Accept", "*/*")
				ctx.Header("Content-Type", "text/event-stream")
				ctx.Writer.Flush()
				return
			}

			if len(response.Message) > 0 {
				select {
				case <-ctx.Request.Context().Done():
					IsClose = true
				default:
					logrus.Info(response.Message)
					if !WriteString(ctx, response.Message, r.IsCompletions) {
						IsClose = true
					}
				}
			}

			if response.Status == vars.Closed && wd {
				WriteDone(ctx, r.IsCompletions)
				IsClose = true
			}
		} else {
			select {
			case <-ctx.Request.Context().Done():
				IsClose = true
			default:
			}
		}
	})
	if !r.Stream {
		if partialResponse.Error != nil {
			responseBingAIError(ctx, partialResponse.Error, r.Stream, r.IsCompletions, token, wd)
			return
		}

		ctx.JSON(200, BuildCompletion(r.IsCompletions, partialResponse.Message))
	}
	store.DeleteMessages(context.Id)
}

func createBingAIConversation(r *cmdtypes.RequestDTO, token string, Isc func() bool) (*types.ConversationContext, error) {
	var (
		id      = "BingAI-" + uuid.NewString()
		bot     string
		model   string
		appId   string
		chain   string
		message string
		preset  string
	)
	switch r.Model {
	case "BingAI":
		bot = vars.Bing
		model = "Sydney"
	default:
		return nil, errors.New(cmdvars.I18n("UNKNOWN_MODEL") + "`" + r.Model + "`")
	}

	var messages []store.Kv
	messages, preset = bingAIMessageConversion(r)

	for idx := len(messages) - 1; idx >= 0; idx-- {
		item := messages[idx]
		if item["author"] == "user" {
			message = item["text"]
			messages = append(messages[:idx], messages[idx+1:]...)
			break
		}
	}

	description := ""
	if l := len(messages); l > vars.BingMaxMessage-2 {
		mergeMessages := messages[0 : l-(vars.BingMaxMessage-4)]

		for _, item := range mergeMessages {
			switch item["author"] {
			case "user":
				description += "Human：" + item["text"] + "\n\n"
			case "bot":
				description += "Assistant：" + item["text"] + "\n\n"
			}
		}

		latelyMessages := messages[l-(vars.BingMaxMessage-4):]
		latelyMessages[0]["text"] = "请改为从此页面回答。\n[使用此页面的对话作为我们之前的对话记录进行后续交流]\n\n" + latelyMessages[0]["text"]
		messages = append([]store.Kv{
			{
				"author":      "user",
				"description": description,
				"contextType": "WebPage",
				"messageType": "Context",
				"sourceName":  "history.md",
				"sourceUrl":   "file:///Users/admin/Desktop/history.md",
				"privacy":     "Internal",
			},
		}, latelyMessages...)
	}

	store.CacheMessages(id, messages)
	if message == "" {
		message = "continue"
	}

	ms := messages
	if len(description) > 0 {
		ms = messages[1:]
	}

	fmt.Println("-----------------------Response-----------------\n",
		"-----------------------「 预设区 」-----------------------\n",
		preset,
		"\n\n\n-----------------------「 history.md 」-----------------------\n",
		description,
		"\n\n\n-----------------------「 对话记录 」-----------------------\n",
		ms,
		"\n\n\n-----------------------「 当前对话 」-----------------------\n",
		message,
		"\n--------------------END-------------------")
	return &types.ConversationContext{
		Id:      id,
		Token:   token,
		Preset:  preset,
		Prompt:  message,
		Bot:     bot,
		Model:   model,
		Proxy:   cmdvars.Proxy,
		AppId:   appId,
		BaseURL: bingBaseURL,
		Chain:   chain,
		H:       handle(Isc),
	}, nil
}

func handle(Isc func() bool) types.CustomCacheHandler {
	return func(rChan any) func(*types.CacheBuffer) error {
		pos := 0
		return func(self *types.CacheBuffer) error {
			partialResponse := rChan.(chan edge.PartialResponse)
			response, ok := <-partialResponse
			if !ok {
				self.Closed = true
				return nil
			}

			if response.Error != nil {
				logrus.Error(response.Error)
				return response.Error
			}

			if Isc() {
				self.Closed = true
				return nil
			}

			if response.Type == 2 {
				if response.Item.Throttling != nil {
					vars.BingMaxMessage = response.Item.Throttling.Max
				}

				messages := response.Item.Messages
				if messages == nil {
					goto label
				}

				for _, value := range *messages {
					if value.Type == "Disengaged" {
						// delete(bot.sessions, ctx.Id)
						if response.Text == "" {
							response.Text = "对不起，我不想继续这个对话。我还在学习中，所以感谢你的理解和耐心。🙏"
						}
					}
				}

			label:
			}

			str := []rune(response.Text)
			length := len(str)
			if pos >= length {
				return nil
			}
			self.Cache += string(str[pos:])
			pos = length
			return nil
		}
	}
}

func bingAIMessageConversion(r *cmdtypes.RequestDTO) ([]store.Kv, string) {
	var messages []store.Kv
	var preset string
	temp := ""
	author := ""

	// 遍历归类
	for _, item := range r.Messages {
		role := item["role"]
		if author == role {
			content := item["content"]
			if content == "[Start a new Chat]" {
				continue
			}
			temp += "\n\n" + content
			continue
		}

		if temp != "" {
			switch author {
			case "system":
				if len(messages) == 0 {
					preset = temp
					author = role
					temp = item["content"]
					continue
				}
				fallthrough
			case "user":
				messages = append(messages, store.Kv{
					"author": "user",
					"text":   temp,
				})
			case "assistant":
				messages = append(messages, store.Kv{
					"author": "bot",
					"text":   temp,
				})
			}
		}

		author = role
		temp = item["content"]
	}

	// 最后一次循环的文本
	if temp != "" {
		_author := ""
		if author == "system" || author == "user" {
			_author = "user"
		} else {
			_author = "bot"
		}
		if l := len(messages); l > 0 && messages[l-1]["author"] == _author {
			if strings.Contains(temp, "<rule>") { // 特殊标记特殊处理
				messages[l-1]["text"] = temp + "\n\n" + messages[l-1]["text"]
			} else {
				messages[l-1]["text"] += "\n\n" + temp
			}
		} else {
			switch _author {
			case "user":
				messages = append(messages, store.Kv{
					"author": "user",
					"text":   temp,
				})
			case "bot":
				messages = append(messages, store.Kv{
					"author": "bot",
					"text":   temp,
				})
			}
		}
	}
	return messages, preset
}

func responseBingAIError(ctx *gin.Context, err error, isStream bool, isCompletions bool, token string, wd bool) {
	errMsg := err.Error()
	if strings.Contains(errMsg, "User needs to solve CAPTCHA to continue") {
		errMsg = "用户需要人机验证...  已尝试自动验证，若重新生成文本无效请手动验证。"
		if strings.Contains(token, "_U=") {
			split := strings.Split(token, ";")
			for _, item := range split {
				if strings.Contains(item, "_U=") {
					token = strings.TrimSpace(strings.ReplaceAll(item, "_U=", ""))
					break
				}
			}
		}
		if e := util.SolveCaptcha(token); e != nil {
			errMsg += "\n\n" + e.Error()
		}
	}
	ResponseError(ctx, errMsg, isStream, isCompletions, wd)
}
