package you

import (
	"errors"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

var (
	Adapter = API{}
	Model   = "you"

	clearance = ""
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0"

	youRollContainer common.RollContainer[string]
)

type API struct {
	plugin.BaseAdapter
}

func init() {
	common.AddInitialized(func() {
		if !pkg.Config.GetBool("you.enabled") {
			return
		}

		cookies := pkg.Config.GetStringSlice("you.cookies")
		if len(cookies) == 0 {
			return
		}
		youRollContainer = common.NewRollContainer[string](cookies)
		youRollContainer.Condition = condition

		port := pkg.Config.GetString("you.helper")
		if port == "" {
			port = "8081"
		}

		you.Exec(port)
		common.AddExited(you.Exit)
	})
}

func (API) Match(_ *gin.Context, model string) bool {
	return strings.HasPrefix(model, "you/")
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
	if youRollContainer.Len() == 0 {
		response.Error(ctx, -1, "empty cookies")
		return
	}

	cookie, err := youRollContainer.Roll()
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	defer resetMarker(cookie)

	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	completion.Model = completion.Model[4:]
	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	chat := you.New(cookie, completion.Model, proxies)
	chat.LimitWithE(true)
	chat.Client(plugin.HTTPClient)

	if err = tryCloudFlare(ctx); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	chat.CloudFlare(clearance, userAgent)
	pMessages, currMessage, tokens, err := mergeMessages(completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	// 清理多余的标签
	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	ctx.Set(ginTokens, tokens)

	ch, err := chat.Reply(common.GetGinContext(ctx), pMessages, currMessage, tokens >= 32*1000)
	if err != nil {
		logger.Error(err)
		var se emit.Error
		if errors.As(err, &se) && se.Code > 400 {
			_ = youRollContainer.SetMarker(cookie, 2)
		}
		response.Error(ctx, -1, err)
		return
	}

	content := waitResponse(ctx, matchers, cancel, ch, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func resetMarker(cookie string) {
	marker, e := youRollContainer.GetMarker(cookie)
	if e != nil {
		logger.Error(e)
		return
	}

	if marker == 0 || marker > 1 {
		return
	}

	e = youRollContainer.SetMarker(cookie, 0)
	if e != nil {
		logger.Error(e)
	}
}

func tryCloudFlare(ctx *gin.Context) error {
	if clearance == "" {
		port := pkg.Config.GetString("you.helper")
		r, err := emit.ClientBuilder(plugin.HTTPClient).
			GET("http://127.0.0.1:"+port+"/clearance").
			DoC(emit.Status(http.StatusOK), emit.IsJSON)
		if err != nil {
			logger.Error(err)
			return err
		}

		obj, err := emit.ToMap(r)
		if err != nil {
			logger.Error(err)
			response.Error(ctx, -1, err)
			return err
		}

		data := obj["data"].(map[string]interface{})
		clearance = data["cookie"].(string)
		userAgent = data["userAgent"].(string)
	}
	return nil
}

func joinMatchers(ctx *gin.Context, matchers []common.Matcher) (chan error, []common.Matcher) {
	// 自定义标记块中断
	cancel, matcher := common.NewCancelMather(ctx)
	matchers = append(matchers, matcher)
	return cancel, matchers
}

func condition(cookie string) bool {
	marker, err := youRollContainer.GetMarker(cookie)
	if err != nil {
		logger.Error(err)
		return false
	}
	return marker == 0
}
