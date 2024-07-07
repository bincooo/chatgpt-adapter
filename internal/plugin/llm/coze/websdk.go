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
	"reflect"
	"sync"
	"time"
)

var (
	w_mu          sync.Mutex
	taskContainer = make([]map[string]interface{}, 0)
)

func addTask(value map[string]interface{}) {
	w_mu.Lock()
	defer w_mu.Unlock()
	taskContainer = append(taskContainer, value)
}

func delTask(value map[string]interface{}) {
	w_mu.Lock()
	defer w_mu.Unlock()
	if len(taskContainer) == 0 {
		return
	}

	for idx := 0; idx < len(taskContainer); idx++ {
		if reflect.DeepEqual(taskContainer[idx], value) {
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

	count := pkg.Config.GetInt("coze.websdk-counter")
	if count > 0 {
		num := counter[cookies.(string)]
		if num >= count {
			_ = cookiesPollContainer.SetMarker(value, 2) // 达到计数数量，进入静置区
			counter[cookies.(string)] = 0                // 重置计数
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
	go _tasks(opts...)
	go _loopTasks()
}

// 重置任务函数
func _loopTasks() {
	s5 := 5 * time.Second
	baseUrl := pkg.Config.GetString("serverless.baseUrl")
	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + pkg.Config.GetString("you.helper")
	}

	for {
		if len(taskContainer) == 0 {
			time.Sleep(s5)
			continue
		}

		container := make([]map[string]interface{}, len(taskContainer))
		copy(container, taskContainer)
		for _, value := range container {
			cookies, ok := value["cookies"]
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
					if _tasks(value) {
						delTask(value)
					}
					continue
				}

				timeout, cancel = context.WithTimeout(context.Background(), 120*time.Second)
				_, err = emit.ClientBuilder(plugin.HTTPClient).
					Context(timeout).
					POST(baseUrl + "/coze/del").
					JHeader().
					Body(value).
					DoS(http.StatusOK)
				cancel()
				if err != nil {
					logger.Errorf("coze websdk 删除失败[%s]：%v", value["email"], err)
					continue
				}
			}

			if _tasks(value) {
				delTask(value)
			}
		}
	}
}

// 初始任务函数
func _tasks(opts ...map[string]interface{}) (exec bool) {
	time.Sleep(6 * time.Second) // 等待程序启动就绪
	baseUrl := pkg.Config.GetString("serverless.baseUrl")
	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + pkg.Config.GetString("you.helper")
	}

	for _, value := range opts {
		timeout, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		payload := make(map[string]interface{})
		copyMap(payload, value)
		payload["bot"] = "custom-128k"
		response, err := emit.ClientBuilder(plugin.HTTPClient).
			Context(timeout).
			POST(baseUrl + "/coze/login").
			JHeader().
			Body(payload).
			DoS(http.StatusOK)
		if err != nil {
			cancel()
			logger.Errorf("coze websdk 同步失败[%s]：%v", value["email"], err)
			taskContainer = append(taskContainer, value)
			continue
		}

		obj, err := emit.ToMap(response)
		cancel()
		if err != nil {
			logger.Errorf("coze websdk 同步失败[%s]：%v", value["email"], err)
			taskContainer = append(taskContainer, value)
			continue
		}

		if v, ok := obj["ok"].(bool); !ok || !v {
			logger.Errorf("coze websdk 同步失败[%s]", value["email"])
			taskContainer = append(taskContainer, value)
			continue
		}

		value["cookies"] = obj["data"]
		cookiesPollContainer.Add(value)
		logger.Infof("coze websdk 同步成功[%s]", value["email"])

		options, _, err := newOptions(vars.Proxies, "coze/websdk", nil)
		if err != nil {
			logger.Error(err)
			return false
		}

		co, msToken := extCookie(obj["data"].(string))
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
		err = chat.Publish(timeout, botId)
		cancel()
		if err != nil {
			logger.Error(err)
			continue
		}

		logger.Info("发布bot成功")
		exec = true
	}

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
