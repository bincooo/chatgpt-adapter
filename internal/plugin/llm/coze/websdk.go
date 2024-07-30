package coze

import (
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"github.com/bincooo/coze-api"
	"github.com/bincooo/emit.io"
	"net/http"
	"sync"
	"time"
)

type obj struct {
	keyv  map[string]interface{}
	count int
}

var (
	w_mu          sync.Mutex
	taskContainer = make([]*obj, 0)

	w_init  = true
	w_retry = 3
)

func addTask(value map[string]interface{}) {
	w_mu.Lock()
	defer w_mu.Unlock()
	taskContainer = append(taskContainer, &obj{value, w_retry})
}

func delTask(value *obj) {
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

func Condition(value map[string]interface{}) bool {
	cookies, ok := value["cookies"]
	if !ok {
		return false
	}

	marker, err := cookiesPollContainer.GetMarker(value)
	if err != nil {
		logger.Error(err)
		return false
	}

	if marker != 0 {
		return false
	}

	count := pkg.Config.GetInt("coze.websdk.counter")
	if count > 0 {
		num := counter[cookies.(string)]
		if num >= count {
			_ = cookiesPollContainer.SetMarker(value, 2) // 达到计数数量，进入静置区
			counter[cookies.(string)] = 0                // 重置计数
			logger.Infof("[coze] 到达计数数量，进入静置区")
			return false
		}
	}

	options, _, err := newOptions(vars.Proxies, "coze/websdk", nil)
	if err != nil {
		logger.Error(err)
		return false
	}

	co, msToken := extCookie(cookies.(string))
	chat := coze.New(co, msToken, options)
	chat.Session(plugin.HTTPClient)

	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	credits, err := chat.QueryWebSdkCredits(timeout)
	if err != nil {
		logger.Error(err)
		return false
	}

	logger.Infof("coze websdk credits[%s]: %v", value["email"], credits)
	if credits == 0 { // 额度用尽，放入重置任务容器中
		cookiesPollContainer.Del(value)
		addTask(value)
	}
	return credits > 0
}

func resetMarker(key interface{}) {
	marker, err := cookiesPollContainer.GetMarker(key)
	if err != nil {
		logger.Error(err)
		return
	}

	if marker != 1 {
		return
	}

	err = cookiesPollContainer.SetMarker(key, 0)
	if err != nil {
		logger.Error(err)
	}
}

func runTasks(opts ...map[string]interface{}) {
	objs := make([]*obj, 0)
	for _, opt := range opts {
		objs = append(objs, &obj{opt, w_retry})
	}
	go initTasks(objs...)
	go loopTasks()
}

// 重置任务函数
func loopTasks() {
	s5 := 5 * time.Second
	baseUrl := pkg.Config.GetString("serverless.baseUrl")
	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + pkg.Config.GetString("you.helper")
	}

	for {
		// 等待初始化完成
		if w_init {
			time.Sleep(s5)
			continue
		}

		if len(taskContainer) == 0 {
			time.Sleep(s5)
			continue
		}

		container := make([]*obj, len(taskContainer))
		copy(container, taskContainer)
		for _, value := range container {
			cookies, ok := value.keyv["cookies"]
			if ok {
				options, _, err := newOptions(vars.Proxies, "coze/websdk", nil)
				if err != nil {
					logger.Error(err)
					continue
				}

				co, msToken := extCookie(cookies.(string))
				chat := coze.New(co, msToken, options)
				chat.Session(plugin.HTTPClient)

				timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				credits, err := chat.QueryWebSdkCredits(timeout)
				cancel()

				if err == nil && credits > 0 {
					logger.Infof("有剩余额度：%d, 不用重置", credits)
					delTask(value)
					if initTasks(value) {
						//
					}
					continue
				}

				timeout, cancel = context.WithTimeout(context.Background(), 120*time.Second)
				response, err := emit.ClientBuilder(plugin.HTTPClient).
					Context(timeout).
					POST(baseUrl + "/coze/del").
					JHeader().
					Body(value.keyv).
					DoS(http.StatusOK)
				cancel()
				if err != nil {
					logger.Errorf("coze websdk 删除失败[%s]：%v", value.keyv["email"], err)
					if emit.IsJSON(response) == nil {
						logger.Error(emit.TextResponse(response))
					}
					if value.count == 0 {
						delTask(value)
					}
					value.count -= 1
					continue
				}
			}

			delTask(value)
			if initTasks(value) {
				//
			}
		}
	}
}

// 初始任务函数
func initTasks(opts ...*obj) (exec bool) {
	time.Sleep(6 * time.Second) // 等待程序启动就绪
	baseUrl := pkg.Config.GetString("serverless.baseUrl")
	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + pkg.Config.GetString("you.helper")
	}

	for _, value := range opts {
		if value.count <= 0 {
			continue
		}

		timeout, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		payload := make(map[string]interface{})
		copyMap(payload, value.keyv)
		payload["bot"] = "custom-128k"
		response, err := emit.ClientBuilder(plugin.HTTPClient).
			Context(timeout).
			POST(baseUrl + "/coze/login").
			JHeader().
			Body(payload).
			DoS(http.StatusOK)
		if err != nil {
			cancel()
			logger.Errorf("coze websdk 同步失败[%s]：%v", value.keyv["email"], err)
			taskContainer = append(taskContainer, &obj{value.keyv, value.count - 1})
			continue
		}

		o, err := emit.ToMap(response)
		cancel()
		if err != nil {
			logger.Errorf("coze websdk 同步失败[%s]：%v", value.keyv["email"], err)
			taskContainer = append(taskContainer, &obj{value.keyv, value.count - 1})
			continue
		}

		if v, ok := o["ok"].(bool); !ok || !v {
			logger.Errorf("coze websdk 同步失败[%s]", value.keyv["email"])
			taskContainer = append(taskContainer, &obj{value.keyv, value.count - 1})
			continue
		}

		value.keyv["cookies"] = o["data"]
		cookiesPollContainer.Add(value.keyv)
		logger.Infof("coze websdk 同步成功[%s]", value.keyv["email"])

		options, _, err := newOptions(vars.Proxies, "coze/websdk", nil)
		if err != nil {
			logger.Error(err)
			w_init = false
			return false
		}

		co, msToken := extCookie(o["data"].(string))
		chat := coze.New(co, msToken, options)
		chat.Session(plugin.HTTPClient)

		timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		bots, err := chat.QueryBots(timeout)
		cancel()
		if err != nil {
			logger.Error(err)
			continue
		}

		botId := ""
		for _, v := range bots {
			info := v.(map[string]interface{})
			if info["name"] == "custom-128k" {
				botId = info["id"].(string)
				break
			}
		}

		if botId == "" {
			logger.Error("custom-128k bot not found")
			continue
		}

		timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		space, err := chat.GetSpace(timeout)
		cancel()
		if err != nil {
			logger.Errorf("custom-128k space err: %v", err)
			continue
		}

		model := pkg.Config.GetString("coze.websdk.model")
		if model == "" {
			model = coze.ModelGpt4o_128k
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
		}, "")
		cancel()
		if err != nil {
			logger.Errorf("custom-128k drafbot err: %v", err)
			continue
		}

		time.Sleep(3 * time.Second)
		timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		err = chat.Publish(timeout, botId, map[string]interface{}{
			"999": map[string]interface{}{
				"sdk_version": "0.1.0-beta.5",
			},
		})
		cancel()
		if err != nil {
			logger.Error(err)
			addTask(value.keyv)
			continue
		}

		logger.Info("发布bot成功")
		exec = true
	}

	w_init = false
	return
}

func copyMap(target map[string]interface{}, src map[string]interface{}) {
	if src == nil || target == nil {
		return
	}

	for k, v := range src {
		target[k] = v
	}
}
