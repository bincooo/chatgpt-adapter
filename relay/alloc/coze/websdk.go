package coze

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/logger"
	"context"
	"github.com/bincooo/coze-api"
	"github.com/bincooo/emit.io"
	"github.com/iocgo/sdk/env"
	"net/http"
	"strings"
	"sync"
	"time"
)

type obj struct {
	value *account
	count int
}

var (
	w_mu          sync.Mutex
	taskContainer = make([]*obj, 0)

	w_init  = true
	w_retry = 3

	bot string
)

func appendTask(value *account) {
	if value == nil {
		return
	}
	w_mu.Lock()
	defer w_mu.Unlock()
	taskContainer = append(taskContainer, &obj{value, w_retry})
}

func removeTask(value *obj) {
	if value == nil {
		return
	}
	w_mu.Lock()
	defer w_mu.Unlock()
	if len(taskContainer) == 0 {
		return
	}

	for idx := 0; idx < len(taskContainer); idx++ {
		if taskContainer[idx] == value {
			taskContainer = append(taskContainer[:idx], taskContainer[idx+1:]...)
			return
		}
	}
}

func condition(proxied string) func(value *account, argv ...interface{}) bool {
	return func(value *account, argv ...interface{}) bool {
		cookies := value.Cookies
		if cookies == "" {
			return false
		}

		marker, err := cookiesContainer.Marked(value)
		if err != nil {
			logger.Error(err)
			return false
		}

		if marker != 0 {
			return false
		}

		options := coze.NewDefaultOptions("xxx", "xxx", 1000, false, proxied)
		co, msToken := extCookie(cookies)
		chat := coze.New(co, msToken, options)
		chat.Session(common.HTTPClient)

		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		credits, err := chat.QueryWebSdkCredits(timeout)
		if err != nil {
			logger.Error(err)
			return false
		}

		logger.Infof("coze websdk credits[%s]: %v", value.E, credits)
		if credits == 0 { // 额度用尽，放入重置任务容器中
			if err = cookiesContainer.Remove(value); err != nil {
				logger.Error(err)
				return false
			}
			appendTask(value)
		}
		return credits > 0
	}
}

func resetMarked(key interface{}) {
	marker, err := cookiesContainer.Marked(key)
	if err != nil {
		logger.Error(err)
		return
	}

	if marker != 1 {
		return
	}

	err = cookiesContainer.MarkTo(key, 0)
	if err != nil {
		logger.Error(err)
	}
}

func run(env *env.Environment, opts ...*account) {
	objs := make([]*obj, 0)
	for _, opt := range opts {
		objs = append(objs, &obj{opt, w_retry})
	}
	go runTasks(env, objs...)
	go loop(env)
}

// 重置任务函数
func loop(env *env.Environment) {
	s5 := 5 * time.Second
	baseUrl := env.GetString("browser-less.reversal")
	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + env.GetString("browser-less.port")
	}

	for {
		// 等待初始化完成
		if w_init || len(taskContainer) == 0 {
			time.Sleep(s5)
			continue
		}

		container := make([]*obj, len(taskContainer))
		copy(container, taskContainer)
		for _, item := range container {
			cookies := item.value.Cookies
			if cookies != "" {
				options := coze.NewDefaultOptions("xxx", "xxx", 1000, false, env.GetString("server.proxied"))
				co, msToken := extCookie(cookies)
				chat := coze.New(co, msToken, options)
				chat.Session(common.HTTPClient)

				timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				credits, err := chat.QueryWebSdkCredits(timeout)
				cancel()

				if err == nil && credits > 0 {
					logger.Infof("有剩余额度：%d, 不用重置", credits)
					removeTask(item)
					if runTasks(env, item) {
						//
					}
					continue
				}

				timeout, cancel = context.WithTimeout(context.Background(), 120*time.Second)
				response, err := emit.ClientBuilder(common.HTTPClient).
					Context(timeout).
					POST(baseUrl + "/coze/del").
					JSONHeader().
					Body(item.value).
					DoS(http.StatusOK)
				cancel()
				if err != nil {
					logger.Errorf("coze websdk 删除失败[%s]：%v", item.value.E, err)
					if response != nil && strings.Contains(response.Header.Get("content-type"), "application/json") {
						logger.Error(emit.TextResponse(response))
					}
					if item.count == 0 {
						removeTask(item)
					}
					item.count -= 1
					continue
				}
			}

			removeTask(item)
			if runTasks(env, item) {
				//
			}
		}
	}
}

// 执行任务函数
func runTasks(env *env.Environment, opts ...*obj) (exec bool) {
	time.Sleep(6 * time.Second) // 等待程序启动就绪
	baseUrl := env.GetString("browser-less.reversal")
	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + env.GetString("browser-less.port")
	}

	system := env.GetString("coze.websdk.system")
	if bot == "" {
		bot = env.GetString("coze.websdk.bot")
		if bot == "" {
			bot = "custom-128k"
		}
	}

	model := env.GetString("coze.websdk.model")
	if model == "" {
		model = coze.ModelGpt4o_128k
	}

	for _, item := range opts {
		if item.count <= 0 {
			continue
		}

		timeout, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		payload := make(map[string]interface{})
		mapCopy(payload, item.value)
		payload["bot"] = bot
		response, err := emit.ClientBuilder(common.HTTPClient).
			Context(timeout).
			POST(baseUrl + "/coze/login").
			JSONHeader().
			Body(payload).
			DoS(http.StatusOK)
		if err != nil {
			cancel()
			logger.Errorf("coze websdk 同步失败[%s]：%v", item.value.E, err)
			taskContainer = append(taskContainer, &obj{item.value, item.count - 1})
			if response != nil && strings.Contains(response.Header.Get("content-type"), "application/json") {
				logger.Error(emit.TextResponse(response))
			}
			continue
		}

		o, err := emit.ToMap(response)
		cancel()
		if err != nil {
			logger.Errorf("coze websdk 同步失败[%s]：%v", item.value.E, err)
			taskContainer = append(taskContainer, &obj{item.value, item.count - 1})
			continue
		}

		if v, ok := o["ok"].(bool); !ok || !v {
			logger.Errorf("coze websdk 同步失败[%s]", item.value.E)
			taskContainer = append(taskContainer, &obj{item.value, item.count - 1})
			continue
		}

		item.value.Cookies = o["data"].(string)
		cookiesContainer.Add(item.value)
		logger.Infof("coze websdk 同步成功[%s]", item.value.E)

		proxied := env.GetString("server.proxied")
		options := coze.NewDefaultOptions("xxx", "xxx", 1000, false, proxied)
		co, msToken := extCookie(o["data"].(string))
		chat := coze.New(co, msToken, options)
		chat.Session(common.HTTPClient)

		timeout, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		bots, err := chat.QueryBots(timeout)
		cancel()
		if err != nil {
			logger.Error(err)
			continue
		}

		botId := ""
		for _, v := range bots {
			info := v.(map[string]interface{})
			if info["name"] == bot {
				botId = info["id"].(string)
				break
			}
		}

		if botId == "" {
			logger.Error(bot + " bot not found")
			continue
		}

		timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		space, err := chat.GetSpace(timeout)
		cancel()
		if err != nil {
			logger.Errorf("%s space err: %v", bot, err)
			continue
		}

		logger.Infof("publish model: %s ...", model)

		maxTokens := 8192
		if model == coze.ModelClaude3Haiku_200k || model == coze.ModelClaude35Sonnet_200k {
			maxTokens = 4096
		}

		chat.Bot(botId, space, 4, true)
		timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		err = chat.DraftBot(context.Background(), coze.DraftInfo{
			Model:       model,
			Temperature: 0.75,
			TopP:        1,
			MaxTokens:   maxTokens,
		}, system)
		cancel()
		if err != nil {
			logger.Errorf("%s drafbot err: %v", bot, err)
			continue
		}

		time.Sleep(3 * time.Second)
		timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		err = chat.Publish(timeout, botId, map[string]interface{}{
			"999": map[string]interface{}{
				"sdk_version": "1.1.0-beta.0",
			},
		})
		cancel()
		if err != nil {
			logger.Error(err)
			appendTask(item.value)
			continue
		}

		logger.Info("发布bot成功")
		exec = true
	}

	w_init = false
	return
}

func mapCopy(target map[string]interface{}, value *account) {
	if value == nil || target == nil {
		return
	}
	H := func(key string, value string) {
		if value != "" {
			target[key] = value
		}
	}
	H("email", value.E)
	H("password", value.P)
	H("validate", value.V)
	H("cookies", value.Cookies)
}
