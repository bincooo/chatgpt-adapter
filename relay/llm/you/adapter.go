package you

import (
	"encoding/json"
	"strings"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

var (
	Model = "you"
)

type api struct {
	inter.BaseAdapter

	env    *env.Environment
	holder *response.ContentHolder
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	token := ctx.GetString("token")
	if !strings.HasPrefix(model, "you/") {
		return
	}

	slice := api.env.GetStringSlice("you.model")
	for _, mod := range append(slice, []string{
		you.GPT_4,
		you.GPT_4_TURBO,
		you.GPT_4o,
		you.GPT_4o_MINI,
		you.OPENAI_O1,
		you.OPENAI_O1_MINI,
		you.CLAUDE_2,
		you.CLAUDE_3_HAIKU,
		you.CLAUDE_3_SONNET,
		you.CLAUDE_3_5_SONNET,
		you.CLAUDE_3_OPUS,
		you.GEMINI_1_0_PRO,
		you.GEMINI_1_5_PRO,
		you.GEMINI_1_5_FLASH,
	}...) {
		if model[4:] == mod {
			password := api.env.GetString("server.password")
			if password != "" && password != token {
				err = response.UnauthorizedError
				return
			}
			ok = true
			return
		}
	}
	return
}

func (api *api) Models() (slice []model.Model) {
	s := api.env.GetStringSlice("you.model")
	for _, mod := range append(s, []string{
		you.GPT_4,
		you.GPT_4_TURBO,
		you.GPT_4o,
		you.GPT_4o_MINI,
		you.OPENAI_O1,
		you.OPENAI_O1_MINI,
		you.CLAUDE_2,
		you.CLAUDE_3_HAIKU,
		you.CLAUDE_3_SONNET,
		you.CLAUDE_3_5_SONNET,
		you.CLAUDE_3_OPUS,
		you.GEMINI_1_0_PRO,
		you.GEMINI_1_5_PRO,
		you.GEMINI_1_5_FLASH,
	}...) {
		slice = append(slice, model.Model{
			Id:      "you/" + mod,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		})
	}
	return
}

func (api *api) HandleMessages(ctx *gin.Context, completion model.Completion) (messages []model.Keyv[interface{}], err error) {
	var (
		toolMessages = toolcall.ExtractToolMessages(&completion)
	)

	if messages, err = api.holder.Handle(ctx, completion); err != nil {
		return
	}
	messages = append(messages, toolMessages...)
	return
}

func (api *api) ToolChoice(ctx *gin.Context) (ok bool, err error) {
	var (
		cookie     = ctx.GetString("token")
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
		echo       = ctx.GetBool(vars.GinEcho)
	)

	if echo {
		fileMessage, chat, query := mergeMessages(ctx, completion)
		echoMessages(ctx, fileMessage, chat, query)
		return
	}

	if toolChoice(ctx, cookie, proxied, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		echo = ctx.GetBool(vars.GinEcho)
	)

	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		token      = ctx.GetString("token")
	)

	completion.Model = completion.Model[4:]
	fileMessage, chatM, message := mergeMessages(ctx, completion)

	if echo {
		echoMessages(ctx, fileMessage, chatM, message)
		return
	}

	chat := you.New(token, completion.Model, proxies)
	chat.LimitWithE(true)
	chat.Client(common.HTTPClient)
	if clearance := ctx.GetString("clearance"); clearance != "" {
		chat.CloudFlare(clearance, ctx.GetString("userAgent"), ctx.GetString("lang"))
	}

	var cancel chan error
	if api.env.GetBool("you.custom") {
		err = chat.Custom(ctx.Request.Context(), "custom-"+completion.Model, "", false)
		if err != nil {
			logger.Error(err)
			response.Error(ctx, -1, err)
			return
		}
	}

	var chats []you.Message
	if i := len(chatM); i > 2 && chatM[0] == '[' && chatM[i-1] == ']' {
		err = json.Unmarshal([]byte(chatM), &chats)
		if err != nil {
			logger.Error(err)
		}
	}

	ch, err := chat.Reply(ctx.Request.Context(), chats, fileMessage, message)
	if err != nil {
		return
	}

	content := waitResponse(ctx, cancel, ch, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
	return
}
