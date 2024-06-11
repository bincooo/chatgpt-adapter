package coze

import (
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/coze-api"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"os"
	"go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"context"
)

type Config struct {
    Models map[string][]ModelConfig `json:"models"`
}

type ModelConfig struct {
    ID         primitive.ObjectID `bson:"_id,omitempty"`
    Cookie     string             `bson:"cookie"`
    Model      string             `bson:"model"`
    Used       int                `bson:"used"`
    StartTime  int64              `bson:"start_time"` // Corrected to use `bson` tag
    Lock       int                `bson:"lock"`
}

func connectToMongoDB() *mongo.Client {
    uri := os.Getenv("MONGODB_URI")
    client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
    if err != nil {
        log.Fatal(err)
    }
    return client
}

func selectAndLockConfig(modelType string) (*ModelConfig, error) {
    client := connectToMongoDB()
    defer func() {
        if err := client.Disconnect(context.TODO()); err != nil {
            log.Fatal(err)
        }
    }()

    collection := client.Database("coze").Collection("bot")

    // Debug print to check the modelType being queried
    log.Printf("Attempting to select and lock config for modelType: %s\n", modelType)

    currentTime := int64(time.Now().Unix())
    filter := bson.M{
		"modelType": modelType,
		"configs.lock": 0,
		"$or": []bson.M{
			{
				"configs.used": bson.M{"$lte": 50},
				"configs.start_time": bson.M{"$lt": currentTime - 86400},
			},
			{
				"configs.used": bson.M{"$lt": 50},
				"configs.start_time": bson.M{"$gte": currentTime - 86400},
			},
		},
	}	
    update := bson.M{
        "$inc": bson.M{"configs.$.used": 1},
        "$set": bson.M{"configs.$.lock": 1},
    }


    opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

    var updatedDoc bson.M
    err := collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&updatedDoc)
    if err != nil {
        // Debug print to log the error if the document is not found or any other error occurs
        log.Printf("Error finding or updating document: %v\n", err)
        return nil, err
    }

    // Convert the BSON document to a JSON string for debugging
    jsonDoc, err := bson.MarshalExtJSON(updatedDoc, false, false)
    if err != nil {
        log.Printf("Error marshalling BSON to JSON: %v\n", err)
    } else {
        // Debug print the JSON representation of the BSON document
        log.Printf("Updated document in JSON format: %s\n", jsonDoc)
    }

    var selectedConfig ModelConfig
    configs, ok := updatedDoc["configs"].(bson.A) // Sử dụng bson.A cho mảng
    if !ok || len(configs) == 0 {
        log.Println("No available configuration for model")
        return nil, nil
    }

    for _, item := range configs {
        config, ok := item.(bson.M)
        if !ok {
            continue
        }
        selectedConfig.ID = updatedDoc["_id"].(primitive.ObjectID)
        if cookie, ok := config["cookie"].(string); ok {
            selectedConfig.Cookie = cookie
        }
        if model, ok := config["model"].(string); ok {
            selectedConfig.Model = model
        }
        if used, ok := config["used"].(int32); ok {
            selectedConfig.Used = int(used)
        }
        if StartTime, ok := config["start_time"].(int64); ok {
            selectedConfig.StartTime = int64(StartTime)
        }
        if lock, ok := config["lock"].(int32); ok {
            selectedConfig.Lock = int(lock)
        }
		if currentTime-selectedConfig.StartTime > 86400 {
            selectedConfig.Used = 1
            selectedConfig.StartTime = currentTime
            // Update the document with the reset used count and new start_time
            if err := updateConfig(selectedConfig); err != nil {
                log.Printf("Error updating config with reset used count: %v\n", err)
            }
        }
        break
    }
	log.Printf("Start_time: %v\n", selectedConfig.StartTime)
    return &selectedConfig, nil
}

func updateConfig(config ModelConfig) error {
    client := connectToMongoDB()
    defer func() {
        if err := client.Disconnect(context.TODO()); err != nil {
            log.Fatal(err)
        }
    }()

    collection := client.Database("coze").Collection("bot")

    filter := bson.M{"_id": config.ID, "configs.model": config.Model}
    update := bson.M{"$set": bson.M{"configs.$": config}}
    _, err := collection.UpdateOne(context.TODO(), filter, update)
    return err
}

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
	})
}

func (API) Match(ctx *gin.Context, model string) bool {
	if Model == model {
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
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	selectedConfig, err := selectAndLockConfig(completion.Model)
    if err != nil {
        logger.Error(err)
        response.Error(ctx, -1, err.Error())
        return
    }

    if selectedConfig == nil {
        response.Error(ctx, -1, "No available configuration for model "+completion.Model)
        return
    }

	cookie = selectedConfig.Cookie
    completion.Model = selectedConfig.Model

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	pMessages, tokens, err := mergeMessages(ctx)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
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

	var lock *common.ExpireLock
	if mode == 'o' {
		l, e := draftBot(ctx, pMessages, chat, completion)
		if e != nil {
			response.Error(ctx, e.Code, e.Err)
			return
		}
		lock = l
	}

	query := ""
	if mode == 'w' {
		query = pMessages[len(pMessages)-1].Content
		chat.WebSdk(chat.TransferMessages(pMessages[:len(pMessages)-1]))
	} else {
		query = coze.MergeMessages(pMessages)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), coze.Text, query)
	if lock != nil {
		lock.Unlock()
		botId := customBotId(completion.Model)
		rmLock(botId)
		logger.Infof("构建完成解锁：%s", botId)
	}

	selectedConfig.Lock = 0 
    if err := updateConfig(*selectedConfig); err != nil {
        logger.Error(err)
    }

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	// 自定义标记块中断
	cancel, matcher := common.NewCancelMather(ctx)
	matchers = append(matchers, matcher)

	content := waitResponse(ctx, matchers, cancel, chatResponse, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

// return true 终止
func draftBot(ctx *gin.Context, pMessages []coze.Message, chat coze.Chat, completion pkg.ChatCompletion) (eLock *common.ExpireLock, emitErr *emit.Error) {
	var system string
	message := pMessages[0]
	if message.Role == "system" {
		system = message.Content
	}

	var value map[string]interface{}
	value, err := chat.BotInfo(ctx.Request.Context())
	if err != nil {
		logger.Error(err)
		return nil, &emit.Error{Code: -1, Err: err}
	}

	// 加锁
	botId := customBotId(completion.Model)
	eLock = newLock(botId)
	if !eLock.Lock(ctx.Request.Context()) {
		// 上锁失败
		logger.Errorf("上锁失败：%s", botId)
		return nil, &emit.Error{Code: http.StatusTooManyRequests, Err: errors.New("too Many Requests")}
	}

	logger.Infof("上锁成功：%s", botId)
	if err = chat.DraftBot(ctx.Request.Context(), coze.DraftInfo{
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
			logger.Infof("using custom coze options: botId = %s, version = %s, scene = %d", values[0], values[1], scene)
			return
		}

		err = e
		return
	}

	if botId8k == "114514" {
		err = errors.New("请配置 coze.8k 后使用")
		return
	}

	if w8k {
		mode = 'w'
	}
	options = coze.NewDefaultOptions(botId8k, version8k, scene8k, false, proxies)
	// 大于7k token 使用 gpt-128k
	if token := calcTokens(pMessages); token > 7000 {
		if botId8k == "114514" {
			err = errors.New("请配置 coze.128k 后使用")
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
