package coze

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/coze-api"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Adapter = API{}
	Model   = "coze"

	// 35-16k
	botId35_16k   = "114514"
	version35_16k = "9527"
	scene35_16k   = 2

	// 8k
	botId8k   = "114514"
	version8k = "9527"
	scene8k   = 2
	w8k       = false

	// 128k
	botId128k   = "114514"
	version128k = "9527"
	scene128k   = 2
	w128k       = false

	mu    sync.Mutex
	rwMus = make(map[string]*common.ExpireLock)

	counter              = make(map[string]int)
	cookiesPollContainer *common.PollContainer[map[string]interface{}]
)

type API struct {
	plugin.BaseAdapter
}

func init() {
	common.AddInitialized(func() {
		config := pkg.Config.GetStringMap("coze.8k")
		if config != nil {
			botId, ok := config["botid"]
			if ok {
				botId8k = fmt.Sprintf("%v", botId)
				version8k = fmt.Sprintf("%v", config["version"])
				scene8k = config["scene"].(int)
				w8k = config["isw"].(bool)
			}
		}

		config = pkg.Config.GetStringMap("coze.128k")
		if config != nil {
			botId, ok := config["botid"]
			if ok {
				botId128k = fmt.Sprintf("%v", botId)
				version128k = fmt.Sprintf("%v", config["version"])
				scene128k = config["scene"].(int)
				w128k = config["isw"].(bool)
			}
		}

		config = pkg.Config.GetStringMap("coze.images")
		if config != nil {
			botId, ok := config["botid"]
			if ok {
				botId35_16k = fmt.Sprintf("%v", botId)
				version35_16k = fmt.Sprintf("%v", config["version"])
				scene35_16k = config["scene"].(int)
			}
		}

		o := pkg.Config.Get("coze.websdk.accounts")
		if slice, ok := o.([]interface{}); ok {
			var values []map[string]interface{}
			for _, value := range slice {
				v := value.(map[string]interface{})
				values = append(values, v)
			}
			cookiesPollContainer = common.NewPollContainer("coze", make([]map[string]interface{}, 0), 60*time.Second) // 报错进入60秒冷却
			cookiesPollContainer.Condition = Condition
			runTasks(values...)
		}
	})
}

func (API) Match(ctx *gin.Context, model string) bool {
	if Model == model {
		return true
	}

	if model == "coze/websdk" {
		return true
	}

	if strings.HasPrefix(model, "coze/") {
		// coze/botId-version-scene
		values := strings.Split(model[5:], "-")
		if len(values) > 2 {
			_, err := strconv.Atoi(values[2])
			return err == nil
		}
	}

	// 检查绘图
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
		{
			Id:      "coze/websdk",
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
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)

		user      = ""
		assistant = ""

		echo = ctx.GetBool(vars.GinEcho)
	)

	{
		keyv, ok := common.GetGinValue[pkg.Keyv[string]](ctx, vars.GinCharSequences)
		if ok {
			user = keyv.GetString("user")
			assistant = keyv.GetString("assistant")
		}

		if user == "" {
			user = "user"
		}
		if assistant == "" {
			assistant = "assistant"
		}
	}

	tor := func(r string) string {
		switch r {
		case "user":
			return user
		case "assistant":
			return assistant
		default:
			return r
		}
	}

	pMessages, tokens, err := mergeMessages(ctx)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	var meta map[string]interface{}
	if completion.Model == "coze/websdk" && !echo {
		meta, err = cookiesPollContainer.Poll()
		if err != nil {
			logger.Error(err)
			response.Error(ctx, -1, err)
			return
		}

		defer resetMarker(meta)
		cookie = meta["cookies"].(string)
		logger.Infof("roll now cookies: %s", cookie)
		count := pkg.Config.GetInt("coze.websdk.counter")
		if count > 0 {
			counter[cookie] += 1
		}

		completion.Model, err = websdkModel(ctx, proxies, cookie)
		if err != nil {
			logger.Error(err)
			response.Error(ctx, -1, err)
			return
		}
	}

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	if echo {
		content := ""
		if common.IsClaude(ctx, cookie, completion.Model) {
			content = pMessages[0].Content
		} else {
			bytes, _ := json.MarshalIndent(pMessages, "", "  ")
			content = string(bytes)
		}
		response.Echo(ctx, completion.Model, content, completion.Stream)
		return
	}

	ctx.Set(ginTokens, tokens)
	options, mode, err := newOptions(proxies, completion.Model, pMessages)
	if err != nil {
		response.Error(ctx, -1, err)
		return
	}

	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)
	chat.Session(plugin.HTTPClient)

	var lock *common.ExpireLock
	if mode == 'o' {
		l, e := draftBot(ctx, pMessages[0], chat, completion)
		if e != nil {
			response.Error(ctx, e.Code, e.Err)
			return
		}
		lock = l
	}

	query := ""
	if mode == 'w' {
		if meta != nil {
			// 不清楚为什么发布的模型会变，先手动检查一下
			if err = validateWebsdkModel(ctx, chat); err != nil {
				logger.Errorf("validate websdkModel error: %v", err)
				response.Error(ctx, -1, err)
				return
			}
		}

		query = pMessages[len(pMessages)-1].Content
		chat.WebSdk(chat.TransferMessages(pMessages[:len(pMessages)-1]))
	} else {
		var newP []coze.Message
		for _, message := range pMessages {
			message.Role = tor(message.Role)
			newP = append(newP, message)
		}
		query = coze.MergeMessages(newP)
		query = query[:len(query)-13] + "<|" + tor("assistant") + "|>"
	}

	chatResponse, err := chat.Reply(common.GetGinContext(ctx), coze.Text, query)
	// 构建完请求即可解锁
	if lock != nil {
		lock.Unlock()
		botId := customBotId(completion.Model)
		rmLock(botId)
		logger.Infof("构建完成解锁：%s", botId)
	}

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		if meta != nil {
			_ = cookiesPollContainer.SetMarker(meta, 2)
			logger.Infof("coze websdk[%s] 进入冷却状态", meta["email"])
		}
		return
	}

	// 自定义标记块中断
	cancel, matcher := common.NewCancelMatcher(ctx)
	matchers = append(matchers, matcher...)

	content := waitResponse(ctx, matchers, cancel, chatResponse, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func validateWebsdkModel(ctx *gin.Context, chat coze.Chat) error {
	model := pkg.Config.GetString("coze.websdk.model")
	if model == "" {
		return nil
	}

	completion := common.GetGinCompletion(ctx)
	timeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	value, err := chat.BotInfo(timeout)
	if err != nil {
		return err
	}
	mod := value["model"].(string)
	if coze.ModelToId(model) == mod {
		return nil
	}

	err = chat.DraftBot(common.GetGinContext(ctx), coze.DraftInfo{
		Model:       model,
		TopP:        completion.TopP,
		Temperature: completion.Temperature,
		MaxTokens:   completion.MaxTokens,
	}, "")
	if err != nil {
		return err
	}

	return chat.Publish(common.GetGinContext(ctx), value["id"].(string), map[string]interface{}{
		"999": map[string]interface{}{
			"sdk_version": "0.1.0-beta.5",
		},
	})
}

func websdkModel(ctx *gin.Context, proxies string, cookie string) (model string, err error) {
	options, _, err := newOptions(proxies, "coze/websdk", nil)
	if err != nil {
		return "", err
	}

	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)
	chat.Session(plugin.HTTPClient)
	bots, err := chat.QueryBots(ctx)
	if err != nil {
		return "", err
	}

	botId := ""
	for _, value := range bots {
		info := value.(map[string]interface{})
		if info["name"] == "custom-128k" {
			botId = info["id"].(string)
			break
		}
	}

	if botId == "" {
		return "", errors.New("custom-128k bot not found")
	}

	space, _ := chat.GetSpace(ctx)
	return "coze/" + botId + "-" + space + "-1000-w", nil
}

// return true 终止
func draftBot(ctx *gin.Context, systemMessage coze.Message, chat coze.Chat, completion pkg.ChatCompletion) (eLock *common.ExpireLock, emitErr *emit.Error) {
	var system string
	if systemMessage.Role == "system" {
		system = systemMessage.Content
	}

	value, err := chat.BotInfo(common.GetGinContext(ctx))
	if err != nil {
		logger.Error(err)
		return nil, &emit.Error{Code: -1, Err: err}
	}

	// 加锁
	botId := customBotId(completion.Model)
	eLock = newLock(botId)
	if !eLock.Lock(common.GetGinContext(ctx)) {
		// 上锁失败
		logger.Errorf("上锁失败：%s", botId)
		return nil, &emit.Error{Code: http.StatusTooManyRequests, Err: errors.New("too Many Requests")}
	}

	logger.Infof("上锁成功：%s", botId)
	if err = chat.DraftBot(common.GetGinContext(ctx), coze.DraftInfo{
		Model:            value["model"].(string),
		TopP:             completion.TopP,
		Temperature:      completion.Temperature,
		MaxTokens:        completion.MaxTokens,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		ResponseFormat:   0,
	}, system); err != nil {
		// 全局配置修改失败，解锁
		eLock.Unlock()
		rmLock(botId)
		logger.Error(fmt.Errorf("全局配置修改失败，解锁：%s， %v", botId, err))
		return nil, &emit.Error{Code: -1, Err: err}
	}
	return eLock, nil
}

func (API) Generation(ctx *gin.Context) {
	var (
		cookie     = ctx.GetString("token")
		proxies    = ctx.GetString("proxies")
		generation = common.GetGinGeneration(ctx)
	)

	// 只绘画用3.5 16k即可
	if botId35_16k == "114514" {
		response.Error(ctx, -1, "请配置 coze.images 后使用")
		return
	}

	options := coze.NewDefaultOptions(botId35_16k, version35_16k, scene35_16k, false, proxies)
	co, msToken := extCookie(cookie)
	chat := coze.New(co, msToken, options)
	chat.Session(plugin.HTTPClient)

	image, err := chat.Images(ctx.Request.Context(), generation.Message)
	if err != nil {
		logger.Error(err)
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

func newLock(token string) *common.ExpireLock {
	mu.Lock()
	defer mu.Unlock()
	if m, ok := rwMus[token]; ok {
		return m
	}

	m := common.NewExpireLock()
	rwMus[token] = m
	return m
}

func rmLock(token string) {
	mu.Lock()
	defer mu.Unlock()
	if m, ok := rwMus[token]; ok {
		if m.IsIdle() {
			delete(rwMus, token)
		}
	}
}

func customBotId(model string) string {
	if strings.HasPrefix(model, "coze/") {
		values := strings.Split(model[5:], "-")
		return values[0]
	}
	return ""
}

func newOptions(proxies string, model string, pMessages []coze.Message) (options coze.Options, mode byte, err error) {
	if model == "coze/websdk" {
		mode = 'w'
		options = coze.NewDefaultOptions("xxx", "xxx", 1000, false, proxies)
		return
	}

	if strings.HasPrefix(model, "coze/") {
		values := strings.Split(model[5:], "-")
		scene, e := strconv.Atoi(values[2])
		if e == nil {
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

		err = logger.WarpError(e)
		return
	}

	if botId8k == "114514" {
		err = logger.WarpError(errors.New("请配置 coze.8k 后使用"))
		return
	}

	if w8k {
		mode = 'w'
	}
	options = coze.NewDefaultOptions(botId8k, version8k, scene8k, false, proxies)
	// 大于7k token 使用 gpt-128k
	if token := calcTokens(pMessages); token > 7000 {
		if botId8k == "114514" {
			err = logger.WarpError(errors.New("请配置 coze.128k 后使用"))
			return
		}

		if w128k {
			mode = 'w'
		}
		options = coze.NewDefaultOptions(botId128k, version128k, scene128k, false, proxies)
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

func isOwner(model string) bool {
	return strings.HasSuffix(model, "-o")
}

func isWebSdk(model string) bool {
	return strings.HasSuffix(model, "-w")
}
