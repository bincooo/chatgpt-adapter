package gemini

import (
	"context"
	"errors"
	"fmt"
	com "github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/v2/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bincooo/goole15"
)

const MODEL = "gemini"
const login = "http://127.0.0.1:8081/v1/login"

var (
	Adapter = API{}
	// TODO clear loop
	gkv = make(map[uint32]cookieOpts)
	mu  sync.Mutex
)

type cookieOpts struct {
	userAgent string
	cookie    string
}

type candidatesResponse struct {
	Candidates []candidate `json:"candidates"`
}

type candidate struct {
	Content struct {
		Role  string                   `json:"role"`
		Parts []map[string]interface{} `json:"parts"`
	} `json:"content"`
	FinishReason string `json:"finishReason"`
	Index        int    `json:"index"`
}

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case "gemini-1.0-pro-latest", "gemini-1.5-pro-latest", "gemini-1.5-flash-latest":
		return true
	default:
		return false
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "gemini-1.0-pro-latest",
			Object:  "model",
			Created: 1686935002,
			By:      "gemini-adapter",
		}, {
			Id:      "gemini-1.5-pro-latest",
			Object:  "model",
			Created: 1686935002,
			By:      "gemini-adapter",
		}, {
			Id:      "gemini-1.5-flash-latest",
			Object:  "model",
			Created: 1686935002,
			By:      "gemini-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	token := ctx.GetString("token")
	if strings.HasPrefix(token, "AIzaSy") {
		complete(ctx)
	} else {
		complete15(ctx)
	}
}

// https://ai.google.dev/models/gemini?hl=zh-cn
func complete(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		completion = com.GetGinCompletion(ctx)
		matchers   = com.GetGinMatchers(ctx)
	)

	newMessages, tokens := mergeMessages(completion.Messages)
	ctx.Set("tokens", tokens)
	r, err := build(ctx.Request.Context(), proxies, cookie, newMessages, completion)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}
	waitResponse(ctx, matchers, r, completion.Stream)
}

// https://ai.google.dev/models/gemini?hl=zh-cn
func complete15(ctx *gin.Context) {
	var (
		token      = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		completion = com.GetGinCompletion(ctx)
		matchers   = com.GetGinMatchers(ctx)
	)

	newMessages, tokens := mergeMessages15(completion.Messages)
	ctx.Set("tokens", tokens)

	// 解析cookie
	sign, auth, key, user, co, err := extCookie15(ctx.Request.Context(), token, proxies)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	opts := goole.NewDefaultOptions(proxies)
	opts.Temperature(completion.Temperature)
	opts.TopP(completion.TopP)
	opts.TopK(completion.TopK)
	h := com.Hash(token)
	if c, ok := gkv[h]; ok {
		opts.UA(c.userAgent)
	}

	chat := goole.New(co, sign, auth, key, user, opts)
	ch, err := chat.Reply(ctx.Request.Context(), newMessages)
	if err != nil {
		code := -1
		errMessage := err.Error()
		if strings.Contains(errMessage, "429 Too Many Requests") {
			code = http.StatusTooManyRequests
		}
		if strings.Contains(errMessage, "500 Internal Server Error") {
			delete(gkv, h) // 尚不清楚 500 错误的原因
		}
		response.Error(ctx, code, err)
		return
	}
	waitResponse15(ctx, matchers, ch, completion.Stream)
}

func extCookie15(ctx context.Context, token, proxies string) (sign, auth, key, user string, cookie string, err error) {
	var opts cookieOpts
	h := com.Hash(token)

	if !strings.Contains(token, "@gmail.com|") {
		// 不走接口获取的token
		opts = cookieOpts{
			cookie: token,
		}
		//
	} else if co, ok := gkv[h]; ok {
		opts = co
		logger.Info("cookie: ", co.cookie)
	} else {
		s := strings.Split(token, "|")
		if len(s) < 4 {
			err = errors.New("invalid token")
			return
		}

		gLogin := pkg.Config.GetString("goole")
		if gLogin == "" {
			gLogin = login
		}

		mu.Lock()
		defer mu.Unlock()

		timeout, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		response, e := emit.ClientBuilder().
			Proxies(proxies).
			POST(gLogin).
			Context(timeout).
			Header("Authorization", s[3]).
			Body(map[string]string{
				"mail":   s[0],
				"cMail":  s[1],
				"passwd": s[2],
			}).
			JHeader().
			DoS(http.StatusOK)
		if e != nil {
			err = fmt.Errorf("fetch cookies failed: %v", e)
			return
		}

		var result map[string]interface{}
		e = emit.ToObject(response, &result)
		if e != nil {
			err = errors.New(fmt.Sprintf("fetch cookies failed: %v", e))
			return
		}

		if !reflect.DeepEqual(result["ok"], true) {
			err = errors.New(fmt.Sprintf("fetch cookies failed: %s", result["message"]))
			return
		}

		opts = cookieOpts{
			userAgent: result["userAgent"].(string),
			cookie:    result["cookies"].(string),
		}
		gkv[h] = opts
	}

	cookie = opts.cookie
	logger.Info("cookie: ", cookie)
	index := strings.Index(cookie, "[sign=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			sign = cookie[index+6 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}

	index = strings.Index(cookie, "[auth=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			auth = cookie[index+6 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}

	index = strings.Index(cookie, "[key=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			key = cookie[index+5 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}

	index = strings.Index(cookie, "[u=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			user = cookie[index+3 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}
	return
}
