package coze

import (
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

var (
	Adapter = API{}
	Model   = "coze"

	// 35-16k
	botId35_16k   = "7353052833752694791"
	version35_16k = "1712016747307"
	scene35_16k   = 2

	// 8k
	botId8k   = "7353047124357365778"
	version8k = "1712645567468"
	scene8k   = 2

	// 128k
	botId128k   = "7353048532129644562"
	version128k = "1712016880672"
	scene128k   = 2
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(ctx *gin.Context, model string) bool {
	if Model == model {
		return true
	}

	token := ctx.GetString("token")
	if model == "dall-e-3" {
		if strings.Contains(token, "msToken=") || strings.Contains(token, "sessionid=") {
			return true
		}
	}
	return false
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		notebook   = ctx.GetBool("notebook")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	pMessages, tokens := mergeMessages(completion.Messages)
	ctx.Set(ginTokens, tokens)
	options := newOptions(proxies, pMessages)
	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)

	query := ""
	if notebook && len(pMessages) > 0 {
		// notebook 模式只取第一条 content
		query = pMessages[0].Content
	} else {
		query = coze.MergeMessages(pMessages)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), query)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	// 自定义标记块中断
	cancel, matcher := common.NewCancelMather(ctx)
	matchers = append(matchers, matcher)

	waitResponse(ctx, matchers, cancel, chatResponse, completion.Stream)
}

func (API) Generation(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		generation = common.GetGinGeneration(ctx)
	)

	// 只绘画用3.5 16k即可
	options := coze.NewDefaultOptions(botId35_16k, version35_16k, scene35_16k, proxies)
	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)
	image, err := chat.Images(ctx.Request.Context(), generation.Message)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	if (generation.Size == "HD" || strings.HasPrefix(generation.Size, "1792x")) && common.HasMfy() {
		v, e := common.Magnify(ctx, image)
		if e != nil {
			logger.Error(e)
		} else {
			image = v
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"styles:": make([]string, 0),
		"data": []map[string]string{
			{"url": image},
		},
	})
}

func newOptions(proxies string, pMessages []coze.Message) (options coze.Options) {
	opts8k := coze.NewDefaultOptions(botId8k, version8k, scene8k, proxies)
	opts128k := coze.NewDefaultOptions(botId128k, version128k, scene128k, proxies)

	options = opts8k
	tokensL := calcTokens(pMessages)
	if tokensL > 7000 { // 大于7k token 使用 gpt-128k
		options = opts128k
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
