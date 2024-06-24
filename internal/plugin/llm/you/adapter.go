package you

import (
	"context"
	"errors"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	mu sync.Mutex

	Adapter = API{}
	Model   = "you"

	clearance = ""
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0"

	youRollContainer *common.PollContainer[string]
)

type API struct {
	plugin.BaseAdapter
}

func init() {
	common.AddInitialized(func() {
		cookies := pkg.Config.GetStringSlice("you.cookies")
		if len(cookies) == 0 {
			return
		}

		youRollContainer = common.NewPollContainer[string](cookies, 30*time.Minute)
		youRollContainer.Condition = Condition

		if pkg.Config.GetBool("serverless.enabled") {
			port := pkg.Config.GetString("you.helper")
			if port == "" {
				port = "8081"
			}
			you.Exec(port, vars.Proxies, os.Stdout, os.Stdout)
			common.AddExited(you.Exit)
		}
	})
}

func (API) Match(_ *gin.Context, model string) bool {
	if strings.HasPrefix(model, "you/") {
		switch model[4:] {
		case you.GPT_4,
			you.GPT_4o,
			you.GPT_4_TURBO,
			you.CLAUDE_2,
			you.CLAUDE_3_HAIKU,
			you.CLAUDE_3_SONNET,
			you.CLAUDE_3_5_SONNET,
			you.CLAUDE_3_OPUS:
			return true
		}
	}
	return false
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "you/" + you.GPT_4,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.GPT_4o,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.GPT_4_TURBO,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_2,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_HAIKU,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_SONNET,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_5_SONNET,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_OPUS,
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

	var (
		retry   = 3
		cookies []string
	)

	defer func(cookies []string) {
		if len(cookies) == 0 {
			return
		}
		for _, value := range cookies {
			resetMarker(value)
		}
	}(cookies)

label:
	retry--
	cookie, err := youRollContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}
	cookies = append(cookies, cookie)

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

	if err = tryCloudFlare(); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	chat.CloudFlare(clearance, userAgent)
	pMessages, currMessage, tokens, err := mergeMessages(ctx, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	ctx.Set(ginTokens, tokens)
	is32 := tokens < 12000

	ch, err := chat.Reply(common.GetGinContext(ctx), pMessages, currMessage, !is32)
	if err != nil {
		logger.Error(err)
		var se emit.Error
		code := -1
		if errors.As(err, &se) && se.Code > 400 {
			_ = youRollContainer.SetMarker(cookie, 2)
			// 403 重定向？？？
			if se.Code == 403 {
				code = 429
				cleanCf()
			}
		}

		if strings.Contains(err.Error(), "ZERO QUOTA") {
			_ = youRollContainer.SetMarker(cookie, 2)
			code = 429
		}

		if retry > 0 {
			goto label
		}
		response.Error(ctx, code, err)
		return
	}

	content := waitResponse(ctx, matchers, cancel, ch, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func cleanCf() {
	mu.Lock()
	clearance = ""
	mu.Unlock()
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

func tryCloudFlare() error {
	if clearance == "" {
		logger.Info("trying cloudflare ...")

		mu.Lock()
		defer mu.Unlock()
		if clearance != "" {
			return nil
		}

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

func Condition(cookie string) bool {
	marker, err := youRollContainer.GetMarker(cookie)
	if err != nil {
		logger.Error(err)
		return false
	}

	if marker != 0 {
		return false
	}

	//return true
	chat := you.New(cookie, you.CLAUDE_2, vars.Proxies)
	chat.Client(plugin.HTTPClient)
	chat.CloudFlare(clearance, userAgent)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 检查可用次数
	count, err := chat.State(ctx)
	if err != nil {
		var se emit.Error
		if errors.As(err, &se) && se.Code == 403 {
			cleanCf()
			_ = tryCloudFlare()
		}
		logger.Error(err)
		return false
	}

	if count == 0 {
		_ = youRollContainer.SetMarker(cookie, 2)
	}
	return count > 0
}
