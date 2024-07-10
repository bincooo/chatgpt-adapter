package you

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"errors"
	"fmt"
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

	lang      = "cn-ZN,cn;q=0.9"
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

		youRollContainer = common.NewPollContainer[string]("you", cookies, 30*time.Minute)
		youRollContainer.Condition = Condition

		if pkg.Config.GetBool("serverless.enabled") {
			port := pkg.Config.GetString("you.helper")
			if port == "" {
				port = "8081"
			}
			you.Exec(port, vars.Proxies, os.Stdout, os.Stdout)
			common.AddExited(you.Exit)
		}

		//go timer()
	})
}

func timer() {
	m10 := 10 * time.Minute

	for {
		time.Sleep(m10)
		if clearance != "" {
			jar, err := emit.NewCookieJar("https://you.com", clearance)
			if err != nil {
				logger.Errorf("检查you.com 5秒盾失败：%v", err)
				continue
			}

			timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err = emit.ClientBuilder(plugin.HTTPClient).
				Context(timeout).
				Proxies(vars.Proxies).
				GET("https://you.com/").
				CookieJar(jar).
				Header("Accept", "application/json, text/plain, */*").
				Header("Accept-Language", lang).
				Header("Referer", "https://you.com/?chatMode=custom").
				Header("Origin", "https://you.com").
				Header("User-Agent", userAgent).
				DoS(http.StatusOK)
			cancel()
			if err == nil {
				continue
			}

			var se emit.Error
			if !errors.As(err, &se) {
				logger.Error("定时器 you.com 过盾检查失败：%v", err)
				continue
			}

			if se.Code == 403 {
				// 需要重新过盾
				clearance = ""
			} else {
				logger.Error("定时器 you.com 过盾检查失败：%v", err)
				continue
			}
		}

		// 尝试过盾
		if err := tryCloudFlare(); err != nil {
			logger.Errorf("you.com 尝试过盾失败：%v", err)
			continue
		}

		logger.Info("定时器执行 you.com 过盾成功")
	}
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
	var (
		retry   = 3
		cookies []string

		echo = ctx.GetBool(vars.GinEcho)
	)

	defer func() {
		for _, value := range cookies {
			resetMarker(value)
		}
	}()

	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	completion.Model = completion.Model[4:]
	chats, message, tokens, err := mergeMessages(ctx, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if echo {
		pM, _ := you.MergeMessages(chats, true)
		response.Echo(ctx, completion.Model, fmt.Sprintf("--------FILE MESSAGE--------:\n%s\n\n\n--------CURR QUESTION--------:\n%s", pM, message), completion.Stream)
		return
	}

	if youRollContainer.Len() == 0 {
		response.Error(ctx, -1, "empty cookies")
		return
	}

label:
	retry--
	cookie, err := youRollContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	cookies = append(cookies, cookie)
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

	chat.CloudFlare(clearance, userAgent, lang)

	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	ctx.Set(ginTokens, tokens)

	messages, err := you.MergeMessages(chats, true)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	ch, err := chat.Reply(common.GetGinContext(ctx), nil, messages, message)
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

func (API) Messages(ctx *gin.Context) {
	var (
		retry   = 3
		cookies []string
	)

	defer func() {
		for _, value := range cookies {
			resetMarker(value)
		}
	}()

	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		//matchers   = common.GetGinMatchers(ctx)
	)

	completion.Model = completion.Model[4:]

	messages := make([]string, 0)
	if completion.System != "" {
		messages = append(messages, completion.System)
	}

	for _, message := range completion.Messages {
		messages = append(messages, message.GetString("content"))
	}

	if youRollContainer.Len() == 0 {
		response.Error(ctx, -1, "empty cookies")
		return
	}

label:
	retry--
	cookie, err := youRollContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	cookies = append(cookies, cookie)
	chat := you.New(cookie, completion.Model, proxies)
	chat.LimitWithE(true)
	chat.Client(plugin.HTTPClient)

	if err = tryCloudFlare(); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	chat.CloudFlare(clearance, userAgent, lang)
	ch, err := chat.Reply(common.GetGinContext(ctx), nil, strings.Join(messages, "\n\n"), " ")
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

	content := waitMessageResponse(ctx, ch)
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

	if marker != 1 {
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
			if emit.IsJSON(r) == nil {
				logger.Error(emit.TextResponse(r))
			}
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
		lang = data["lang"].(string)
	}
	return nil
}

func joinMatchers(ctx *gin.Context, matchers []common.Matcher) (chan error, []common.Matcher) {
	// 自定义标记块中断
	keyv, ok := common.GetGinValue[pkg.Keyv[string]](ctx, vars.GinCharSequences)
	if ok {
		if user := keyv.GetString("user"); user == "" {
			keyv.Set("user", "\n\nHuman:")
			ctx.Set(vars.GinCharSequences, keyv)
		}
	}

	cancel, matcher := common.NewCancelMatcher(ctx)
	matchers = append(matchers, matcher...)
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
	chat.CloudFlare(clearance, userAgent, lang)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 检查可用次数
	count, err := chat.State(ctx)
	if err != nil {
		var se emit.Error
		if errors.As(err, &se) {
			if se.Code == 403 {
				cleanCf()
				_ = tryCloudFlare()
			}
			if se.Code == 401 { // cookie 失效？？？
				_ = youRollContainer.SetMarker(cookie, 2)
			}
		}
		logger.Error(err)
		return false
	}

	if count == 0 {
		_ = youRollContainer.SetMarker(cookie, 2)
	}
	return count > 0
}
