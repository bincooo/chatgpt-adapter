package hf

import (
	"chatgpt-adapter/internal/agent"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	Adapter  = API{}
	ginSpace = "__prodia_space__"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(ctx *gin.Context, model string) bool {
	if model != "dall-e-3" {
		return false
	}

	token := ctx.GetString("token")
	if token == "sk-prodia-sd" {
		ctx.Set(ginSpace, "prodia-sd")
		return true
	}

	if token == "sk-prodia-xl" {
		ctx.Set(ginSpace, "prodia-xl")
		return true
	}

	if token == "sk-google-xl" {
		ctx.Set(ginSpace, "google")
		return true
	}

	if token == "sk-dalle-4k" {
		ctx.Set(ginSpace, "dalle-4k")
		return true
	}

	if token == "sk-dalle-3-xl" {
		ctx.Set(ginSpace, "dalle-3xl")
		return true
	}

	if token == "sk-animagine-xl-3.1" {
		ctx.Set(ginSpace, "animagine-xl-3.1")
		return true
	}

	return false
}

func (API) Generation(ctx *gin.Context) {
	var (
		value        = ""
		modelSlice   []string
		samplesSlice []string
		space        = ctx.GetString(ginSpace)
		generation   = common.GetGinGeneration(ctx)
	)

	message, err := completeTagsGenerator(ctx, generation.Message)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	message = strings.TrimSpace(message)
	message = strings.ReplaceAll(message, "。", "")
	message = strings.ReplaceAll(message, ".", "")
	message = strings.ReplaceAll(message, "\n", "")
	model := matchModel(generation.Style, space)
	samples := matchSamples(generation.Quality, space)

	logger.Infof("curr space info[%s]: %s, %s", space, model, samples)
	switch space {
	case "prodia-xl":
		modelSlice = xlModels
		samplesSlice = xlSamples
		value, err = Ox001(ctx, model, samples, message)
	case "dalle-4k":
		modelSlice = dalle4kModels
		value, err = Ox002(ctx, model, message)
	case "dalle-3xl":
		value, err = Ox003(ctx, message)
	case "animagine-xl-3.1":
		modelSlice = animagineXl31Models
		samplesSlice = animagineXl31Samples
		value, err = Ox004(ctx, model, samples, message)
	case "google":
		modelSlice = googleModels
		value, err = google(ctx, model, message)
	default:
		modelSlice = sdModels
		samplesSlice = sdSamples
		value, err = Ox000(ctx, model, samples, message)
	}

	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if (generation.Size == "HD" || strings.HasPrefix(generation.Size, "1792x")) && common.HasMfy() {
		v, e := common.Magnify(ctx, value)
		if e != nil {
			logger.Error(e)
		} else {
			value = v
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"styles":  modelSlice,
		"samples": samplesSlice,
		"data": []map[string]string{
			{"url": value},
		},
		"prompt":      message + ", {{{{by famous artist}}}, beautiful, masterpiece, 4k",
		"currStyle":   model,
		"currSamples": samples,
	})
}

func matchSamples(samples, spase string) string {
	switch spase {
	case "prodia-xl":
		if common.Contains(xlSamples, samples) {
			return samples
		}
		return "Euler a"
	case "dalle-3xl":
		return "none"
	case "animagine-xl-3.1":
		if common.Contains(animagineXl31Samples, samples) {
			return samples
		}
		return "Euler a"
	default:
		if common.Contains(sdSamples, samples) {
			return samples
		}
		return "Euler a"
	}
}

func matchModel(style, spase string) string {
	switch spase {
	case "prodia-xl":
		if common.Contains(xlModels, style) {
			return style
		}
		return xlModels[rand.Intn(len(xlModels))]

	case "dalle-4k":
		if common.Contains(dalle4kModels, style) {
			return style
		}
		return dalle4kModels[rand.Intn(len(dalle4kModels))]

	case "google":
		if common.Contains(googleModels, style) {
			return style
		}
		return googleModels[rand.Intn(len(googleModels))]

	case "dalle-3xl":
		return "none"

	case "animagine-xl-3.1":
		if common.Contains(animagineXl31Models, style) {
			return style
		}
		return animagineXl31Models[rand.Intn(len(animagineXl31Models))]

	default:
		if common.Contains(sdModels, style) {
			return style
		}
		return sdModels[rand.Intn(len(sdModels))]
	}
}

func completeTagsGenerator(ctx *gin.Context, content string) (string, error) {
	var (
		proxies = ctx.GetString("proxies")
		model   = pkg.Config.GetString("llm.model")
		cookie  = pkg.Config.GetString("llm.token")
		baseUrl = pkg.Config.GetString("llm.baseUrl")
	)

	c := regexp.MustCompile("<tag content=\"([^>]+)\"\\s?/>")
	matched := c.FindAllStringSubmatch(content, -1)
	var contents []string
	if len(matched) > 0 {
		for _, slice := range matched {
			content = strings.Replace(content, slice[0], "", -1)
			contents = append(contents, slice[1])
		}
	}

	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return strings.Join(contents, ", "), nil
	}

	if strings.Contains(content, "<tag llm=false />") {
		contents = append(contents, strings.Replace(content, "<tag llm=false />", "", -1))
		return strings.Join(contents, ", "), nil
	}

	prefix := ""
	if model == "bing" {
		// prefix += "<pad />"
		//prefix += "<notebook />"
	}

	w := prefix + agent.SDWords
	if ctx.GetString(ginSpace) == "dalle-4k" || ctx.GetString(ginSpace) == "dalle-3xl" {
		w = prefix + agent.SD2Words
	}

	obj := map[string]interface{}{
		"model":  model,
		"stream": false,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": strings.Replace(w, "{{content}}", content, -1),
			},
		},
		"temperature": .8,
		"max_tokens":  4096,
	}

	res, err := fetch(common.GetGinContext(ctx), proxies, baseUrl, cookie, obj)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var r pkg.ChatResponse
	if err = json.Unmarshal(data, &r); err != nil {
		logger.Error("data: %s", data)
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		if r.Error != nil {
			return "", errors.New(r.Error.Message)
		} else {
			return "", errors.New(res.Status)
		}
	}

	message := strings.TrimSpace(r.Choices[0].Message.Content)
	left := strings.Index(message, `"""`)
	right := strings.LastIndex(message, `"""`)

	if left > -1 && left < right {
		message = strings.ReplaceAll(message[left+3:right], "\"", "")
		contents = append(contents, message)
		logger.Infof("system assistant generate message[%s]: %s", model, strings.Join(contents, ", "))
		return strings.Join(contents, ", "), nil
	}

	if strings.HasSuffix(message, `"""`) { // 哎。bing 偶尔会漏掉前面的"""
		message = strings.ReplaceAll(message[:len(message)-3], "\"", "")
		contents = append(contents, message)
		logger.Infof("system assistant generate message[%s]: %s", model, strings.Join(contents, ", "))
		return strings.Join(contents, ", "), nil
	}

	left = strings.Index(message, "```")
	right = strings.LastIndex(message, "```")

	if left > -1 && left < right {
		message = strings.ReplaceAll(message[left+3:right], "\"", "")
		contents = append(contents, message)
		logger.Infof("system assistant generate message[%s]: %s", model, strings.Join(contents, ", "))
		return strings.Join(contents, ", "), nil
	}

	logger.Info("response content: ", message)
	logger.Errorf("system assistant generate message[%s] error: system assistant generate message failed", model)
	return "", errors.New("system assistant generate message failed")
}

func fetch(ctx context.Context, proxies, baseUrl, cookie string, obj interface{}) (*http.Response, error) {
	if strings.Contains(baseUrl, "127.0.0.1") || strings.Contains(baseUrl, "localhost") {
		proxies = ""
	}

	return emit.ClientBuilder(nil).
		Context(ctx).
		Proxies(proxies).
		POST(fmt.Sprintf("%s/v1/chat/completions", baseUrl)).
		Header("Authorization", cookie).
		JHeader().
		Body(obj).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
}
