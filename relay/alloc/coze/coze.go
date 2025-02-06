package coze

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/coze-api"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/proxy"
)

type account struct {
	Cookies string `mapstructure:"-" json:"-"`

	E string `mapstructure:"email" json:"email"`
	P string `mapstructure:"password" json:"password"`
	V string `mapstructure:"validate" json:"validate"`
}

var (
	cookiesContainer *common.PollContainer[*account]
)

func init() {
	inited.AddInitialized(func(env *env.Environment) {
		var values []*account
		err := env.UnmarshalKey("coze.websdk.accounts", &values)
		if err != nil {
			panic(err)
		}
		if len(values) == 0 {
			return
		}

		if !env.GetBool("browser-less.enabled") && env.GetString("browser-less.reversal") == "" {
			panic("don't used browser-less, please setting `browser-less.enabled` or `browser-less.reversal`")
		}

		cookiesContainer = common.NewPollContainer("coze", make([]*account, 0), 60*time.Second) // 报错进入60秒冷却
		cookiesContainer.Condition = condition(env.GetString("server.proxied"))
		run(env, values...)
	})
}

func InvocationHandler(ctx *proxy.Context) {
	var (
		context    = ctx.In[0].(*gin.Context)
		completion = common.GetGinCompletion(context)
		proxied    = env.Env.GetString("server.proxied")
		echo       = context.GetBool(vars.GinEcho)
	)

	if echo || ctx.Method != "Completion" && ctx.Method != "ToolChoice" {
		ctx.Do()
		return
	}

	logger.Infof("execute static proxy [relay/llm/coze.api]: func %s(...)", ctx.Method)

	var (
		err  error
		meta *account

		cookies string
	)

	if isSdk(context, completion.Model) {
		meta, err = cookiesContainer.Poll()
		if err != nil {
			logger.Error(err)
			response.Error(context, -1, err)
			return
		}

		defer resetMarked(meta)
		cookies = meta.Cookies
		logger.Infof("roll now Cookies: %s", cookies)

		completion.Model, err = sdkModel(context, proxied, cookies)
		if err != nil {
			logger.Error(err)
			response.Error(context, -1, err)
			return
		}
		context.Set(vars.GinCompletion, completion)
	}

	values := strings.Split(completion.Model[5:], "-")
	if isOwner(completion.Model) && len(values) > 2 {
		var scene int
		if scene, err = strconv.Atoi(values[2]); err != nil {
			logger.Error(err)
			response.Error(context, -1, err)
			return
		}

		co, msToken := extCookie(cookies)
		options := coze.NewDefaultOptions(values[0], values[1], scene, true, proxied)
		chat := coze.New(co, msToken, options)
		chat.Session(common.HTTPClient)
		emitErr := draftBot(context, "", chat, completion)
		if emitErr != nil {
			response.Error(context, emitErr.Code, emitErr.Err)
			return
		}
	}

	context.Set("token", cookies)

	ctx.Do()

	if ctx.Method == "Completion" {
		err = elseOf[error](ctx.Out[0])
	}
	if ctx.Method == "ToolChoice" {
		err = elseOf[error](ctx.Out[1])
	}

	if err != nil {
		if meta != nil {
			_ = cookiesContainer.MarkTo(meta, 2)
			logger.Infof("coze websdk[%s] 进入冷却状态", meta.E)
		}
		return
	}
}

func isSdk(ctx *gin.Context, model string) bool {
	if common.IsGinCozeWebsdk(ctx) {
		return true
	}
	if model == "coze/websdk" {
		ctx.Set(vars.GinCozeWebsdk, true)
		return true
	}
	return false
}

func sdkModel(ctx *gin.Context, proxies string, cookie string) (model string, err error) {
	options := coze.NewDefaultOptions("xxx", "xxx", 1000, false, proxies)
	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)
	chat.Session(common.HTTPClient)
	bots, err := chat.QueryBots(ctx)
	if err != nil {
		return "", err
	}

	botId := ""
	botn := bot
	if botn == "" {
		botn = "custom-128k"
	}

	for _, value := range bots {
		info := value.(map[string]interface{})
		if info["name"] == botn {
			botId = info["id"].(string)
			break
		}
	}

	if botId == "" {
		return "", errors.New(botn + " bot not found")
	}

	space, _ := chat.GetSpace(ctx)
	return "coze/" + botId + "-" + space + "-1000-w", nil
}

// return true 终止
func draftBot(ctx *gin.Context, systemMessage string, chat coze.Chat, completion model.Completion) (emitErr *emit.Error) {
	value, err := chat.BotInfo(ctx.Request.Context())
	if err != nil {
		logger.Error(err)
		return &emit.Error{Code: -1, Err: err}
	}

	botId := customBotId(completion.Model)
	if err = chat.DraftBot(ctx.Request.Context(), coze.DraftInfo{
		Model:            value["model"].(string),
		TopP:             completion.TopP,
		Temperature:      completion.Temperature,
		MaxTokens:        completion.MaxTokens,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		ResponseFormat:   0,
	}, systemMessage); err != nil {
		logger.Error(fmt.Errorf("全局配置修改失败[%s]：%v", botId, err))
		return &emit.Error{Code: -1, Err: err}
	}
	return
}

func extCookie(co string) (cookie, msToken string) {
	cookie = co
	index := strings.Index(cookie, "[msToken=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			msToken = cookie[index+6 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}
	return
}

func customBotId(model string) string {
	if strings.HasPrefix(model, "coze/") {
		values := strings.Split(model[5:], "-")
		return values[0]
	}
	return ""
}

func isOwner(model string) bool { return strings.HasSuffix(model, "-o") }
func elseOf[T any](obj any) (zero T) {
	if obj == nil {
		return
	}
	return obj.(T)
}
