package coze

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/stream"
)

var (
	Model = "coze"
)

type api struct {
	inter.BaseAdapter

	env *env.Environment
}

func (api *api) Match(ctx *gin.Context, model string) (ok bool, err error) {
	if Model == model {
		ok = true
		return
	}

	var token = ctx.GetString("token")
	if model == "coze/websdk" {
		password := api.env.GetString("server.password")
		if password != "" && password != token {
			err = response.UnauthorizedError
			return
		}
		ok = true
		return
	}

	if strings.HasPrefix(model, "coze/") {
		// coze/botId-version-scene
		values := strings.Split(model[5:], "-")
		if len(values) > 2 {
			_, err = strconv.Atoi(values[2])
			logger.Warn(err)
			ok = err == nil
			return
		}
	}

	// 检查绘图
	if model == "dall-e-3" {
		if strings.Contains(token, "msToken=") || strings.Contains(token, "sessionid=") {
			ok = true
			return
		}
	}
	return
}

func (*api) Models() []model.Model {
	return []model.Model{
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "coze/websdk",
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
	}
}

func (api *api) ToolChoice(ctx *gin.Context) (ok bool, err error) {
	var (
		cookie     = ctx.GetString("token")
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	if toolChoice(ctx, cookie, proxied, completion) {
		ok = true
	}
	return
}

func (api *api) Completion(ctx *gin.Context) (err error) {
	var (
		cookie     = ctx.GetString("token")
		proxied    = api.env.GetString("server.proxied")
		completion = common.GetGinCompletion(ctx)
	)

	newMessages, err := mergeMessages(ctx)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		err = nil
		return
	}

	options, mode, err := newOptions(proxied, completion.Model)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)
	chat.Session(common.HTTPClient)

	query := ""
	if mode == 'w' {
		query = newMessages[len(newMessages)-1].Content
		chat.WebSdk(chat.TransferMessages(newMessages[:len(newMessages)-1]))
	} else {
		query = strings.Join(stream.Map(
			stream.OfSlice(newMessages), func(t coze.Message) string { return t.Content }).ToSlice(), "\n\n")
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), coze.Text, query)
	if err != nil {
		logger.Error(err)
		return
	}

	content := waitResponse(ctx, chatResponse, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
	return
}

func (api *api) Generation(ctx *gin.Context) (err error) {
	var (
		cookie     = ctx.GetString("token")
		proxied    = api.env.GetString("server.proxied")
		generation = common.GetGinGeneration(ctx)
	)

	options, _, err := newOptions(proxied, generation.Model)
	if err != nil {
		return
	}

	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)
	chat.Session(common.HTTPClient)

	image, err := chat.Images(ctx.Request.Context(), generation.Message)
	if err != nil {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"styles:": make([]string, 0),
		"data": []map[string]string{
			{"url": image},
		},
	})

	return
}

func newOptions(proxies string, model string) (options coze.Options, mode byte, err error) {
	if model == "coze/websdk" {
		mode = 'w'
		options = coze.NewDefaultOptions("xxx", "xxx", 1000, false, proxies)
		return
	}

	if strings.HasPrefix(model, "coze/") {
		var scene int
		values := strings.Split(model[5:], "-")
		if scene, err = strconv.Atoi(values[2]); err == nil {
			isO := isOwner(model)
			if isO {
				mode = 'o'
			} else if isWebSdk(model) {
				mode = 'w'
			}
			options = coze.NewDefaultOptions(values[0], values[1], scene, isO, proxies)
			logger.Infof("using custom coze options: botId = %s, version = %s, scene = %d, mode = %c", values[0], values[1], scene, mode)
			return
		}
	}

	err = fmt.Errorf("coze model '%s' is incorrect", model)
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

func isOwner(model string) bool {
	return strings.HasSuffix(model, "-o")
}

func isWebSdk(model string) bool {
	return strings.HasSuffix(model, "-w")
}
