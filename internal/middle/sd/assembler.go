package sd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/agent"
	com "github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	emits "github.com/bincooo/gio.emits"
	"github.com/bincooo/gio.emits/common"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	EMPTRY_EVENT_RETURN map[string]interface{} = nil
)

var (
	sdModels = []string{
		"3Guofeng3_v34.safetensors [50f420de]",
		"absolutereality_V16.safetensors [37db0fc3]",
		"absolutereality_v181.safetensors [3d9d4d2b]",
		"amIReal_V41.safetensors [0a8a2e61]",
		"analog-diffusion-1.0.ckpt [9ca13f02]",
		"anythingv3_0-pruned.ckpt [2700c435]",
		"anything-v4.5-pruned.ckpt [65745d25]",
		"anythingV5_PrtRE.safetensors [893e49b9]",
		"AOM3A3_orangemixs.safetensors [9600da17]",
		"blazing_drive_v10g.safetensors [ca1c1eab]",
		"breakdomain_I2428.safetensors [43cc7d2f]",
		"breakdomain_M2150.safetensors [15f7afca]",
		"cetusMix_Version35.safetensors [de2f2560]",
		"childrensStories_v13D.safetensors [9dfaabcb]",
		"childrensStories_v1SemiReal.safetensors [a1c56dbb]",
		"childrensStories_v1ToonAnime.safetensors [2ec7b88b]",
		"Counterfeit_v30.safetensors [9e2a8f19]",
		"cuteyukimixAdorable_midchapter3.safetensors [04bdffe6]",
		"cyberrealistic_v33.safetensors [82b0d085]",
		"dalcefo_v4.safetensors [425952fe]",
		"deliberate_v2.safetensors [10ec4b29]",
		"deliberate_v3.safetensors [afd9d2d4]",
		"dreamlike-anime-1.0.safetensors [4520e090]",
		"dreamlike-diffusion-1.0.safetensors [5c9fd6e0]",
		"dreamlike-photoreal-2.0.safetensors [fdcf65e7]",
		"dreamshaper_6BakedVae.safetensors [114c8abb]",
		"dreamshaper_7.safetensors [5cf5ae06]",
		"dreamshaper_8.safetensors [9d40847d]",
		"edgeOfRealism_eorV20.safetensors [3ed5de15]",
		"EimisAnimeDiffusion_V1.ckpt [4f828a15]",
		"elldreths-vivid-mix.safetensors [342d9d26]",
		"epicphotogasm_xPlusPlus.safetensors [1a8f6d35]",
		"epicrealism_naturalSinRC1VAE.safetensors [90a4c676]",
		"epicrealism_pureEvolutionV3.safetensors [42c8440c]",
		"ICantBelieveItsNotPhotography_seco.safetensors [4e7a3dfd]",
		"indigoFurryMix_v75Hybrid.safetensors [91208cbb]",
		"juggernaut_aftermath.safetensors [5e20c455]",
		"lofi_v4.safetensors [ccc204d6]",
		"lyriel_v16.safetensors [68fceea2]",
		"majicmixRealistic_v4.safetensors [29d0de58]",
		"mechamix_v10.safetensors [ee685731]",
		"meinamix_meinaV9.safetensors [2ec66ab0]",
		"meinamix_meinaV11.safetensors [b56ce717]",
		"neverendingDream_v122.safetensors [f964ceeb]",
		"openjourney_V4.ckpt [ca2f377f]",
		"pastelMixStylizedAnime_pruned_fp16.safetensors [793a26e8]",
		"portraitplus_V1.0.safetensors [1400e684]",
		"protogenx34.safetensors [5896f8d5]",
		"Realistic_Vision_V1.4-pruned-fp16.safetensors [8d21810b]",
		"Realistic_Vision_V2.0.safetensors [79587710]",
		"Realistic_Vision_V4.0.safetensors [29a7afaa]",
		"Realistic_Vision_V5.0.safetensors [614d1063]",
		"redshift_diffusion-V10.safetensors [1400e684]",
		"revAnimated_v122.safetensors [3f4fefd9]",
		"rundiffusionFX25D_v10.safetensors [cd12b0ee]",
		"rundiffusionFX_v10.safetensors [cd4e694d]",
		"sdv1_4.ckpt [7460a6fa]",
		"v1-5-pruned-emaonly.safetensors [d7049739]",
		"v1-5-inpainting.safetensors [21c7ab71]",
		"shoninsBeautiful_v10.safetensors [25d8c546]",
		"theallys-mix-ii-churned.safetensors [5d9225a4]",
		"timeless-1.0.ckpt [7c4971d4]",
		"toonyou_beta6.safetensors [980f6b15]",
	}

	xlModels = []string{
		"dreamshaperXL10_alpha2.safetensors [c8afe2ef]",
		"dynavisionXL_0411.safetensors [c39cc051]",
		"juggernautXL_v45.safetensors [e75f5471]",
		"realismEngineSDXL_v10.safetensors [af771c3f]",
		"sd_xl_base_1.0.safetensors [be9edd61]",
		"sd_xl_base_1.0_inpainting_0.1.safetensors [5679a81a]",
		"turbovisionXL_v431.safetensors [78890989]",
	}
)

func Generation(ctx *gin.Context, req gpt.ChatGenerationRequest) {
	var (
		index   = 0
		baseUrl = "https://prodia-fast-stable-diffusion.hf.space"
		domain  = pkg.Config.GetString("domain")
		proxies = ctx.GetString("proxies")
		space   = ctx.GetString("prodia.space")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	prompt, err := completeTagsGenerator(ctx, req.Prompt)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	var c *emits.Emits
	hash := emits.SessionHash()
	value := ""
	var eventError error

	var models []string
	model := convertToModel(req.Style, space)
	negativePrompt := "(deformed eyes, nose, ears, nose, leg, head), bad anatomy, ugly"
	params := []interface{}{
		prompt + ", {{{{by famous artist}}}, beautiful, 4k",
		negativePrompt,
		model,
		25,
		"Euler a",
		10,
		1024,
		1024,
		-1,
	}

	switch space {
	case "xl":
		models = xlModels
		baseUrl = "wss://prodia-sdxl-stable-diffusion-xl.hf.space"

		conn, e := common.SocketBuilder().
			Proxies(proxies).
			URL(baseUrl + "/queue/join").
			DoWith(http.StatusSwitchingProtocols)
		if e != nil {
			middle.ResponseWithE(ctx, -1, e)
			return
		}

		c, err = emits.New(ctx.Request.Context(), conn)
		if err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}

	default:
		models = sdModels
		response, e := common.ClientBuilder().
			Context(ctx.Request.Context()).
			Proxies(proxies).
			GET(baseUrl+"/queue/join").
			Query("fn_index", strconv.Itoa(index)).
			Query("session_hash", hash).
			DoWith(http.StatusOK)
		if e != nil {
			middle.ResponseWithE(ctx, -1, e)
			return
		}

		c, err = emits.New(ctx.Request.Context(), response)
		if err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}
	}

	c.Event("send_hash", func(j emits.JoinCompleted) interface{} {
		return map[string]interface{}{
			"fn_index":     index,
			"session_hash": hash,
		}
	})

	c.Event("send_data", func(j emits.JoinCompleted) interface{} {
		obj := map[string]interface{}{
			"data":         params,
			"event_data":   nil,
			"fn_index":     index,
			"session_hash": hash,
			"event_id":     j.EventId,
			"trigger_id":   rand.Intn(15) + 5,
		}
		switch space {
		case "xl":
			return obj
		default:
			_, err = common.ClientBuilder().
				Proxies(proxies).
				Context(ctx.Request.Context()).
				POST(baseUrl + "/queue/data").
				JHeader().
				Body(obj).
				DoWith(http.StatusOK)
			if err != nil {
				eventError = err
			}
			return EMPTRY_EVENT_RETURN
		}
	})

	c.Event("process_completed", func(j emits.JoinCompleted) interface{} {
		d := j.Output.Data
		if len(d) > 0 {
			switch space {
			case "xl":
				file, e := com.SaveBase64(d[0].(string), "png")
				if e != nil {
					eventError = fmt.Errorf("image save failed: %s", j.InitialBytes)
					return EMPTRY_EVENT_RETURN
				}
				value = fmt.Sprintf("%s/file/%s", domain, file)
			default:
				result := d[0].(map[string]interface{})
				value, err = com.Download(proxies, fmt.Sprintf("%s/file=%s", baseUrl, result["path"].(string)), "png")
				if err != nil {
					eventError = err
				}
				value = fmt.Sprintf("%s/file/%s", domain, value)
			}
		} else {
			eventError = fmt.Errorf("image generate failed: %s", j.InitialBytes)
		}
		return EMPTRY_EVENT_RETURN
	})

	if err = middle.IsCanceled(ctx.Request.Context()); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if err = c.Do(); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if eventError != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if (req.Size == "HD" || strings.HasPrefix(req.Size, "1792x")) && com.HasMfy() {
		v, e := com.Magnify(ctx, value)
		if e != nil {
			logrus.Error(e)
		} else {
			value = v
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"styles":  models,
		"data": []map[string]string{
			{"url": value},
		},
		"prompt":    prompt + ", {{{{by famous artist}}}, beautiful, masterpiece, 4k",
		"currStyle": model,
	})
}

func convertToModel(style, spase string) string {
	switch spase {
	case "xl":
		if com.Contains(xlModels, style) {
			return style
		}
		return xlModels[rand.Intn(len(xlModels))]
	default:
		if com.Contains(sdModels, style) {
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

	prefix := "<debug />"
	if model == "bing" {
		prefix += "<pad />"
	}

	obj := map[string]interface{}{
		"model":  model,
		"stream": false,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": strings.Replace(prefix+agent.SDWords, "{{content}}", content, -1),
			},
		},
		"temperature": .8,
		"max_tokens":  4096,
	}

	marshal, _ := json.Marshal(obj)
	response, err := fetch(ctx.Request.Context(), proxies, baseUrl, cookie, marshal)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var r gpt.ChatCompletionResponse
	if err = json.Unmarshal(data, &r); err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		if r.Error != nil {
			return "", errors.New(r.Error.Message)
		} else {
			return "", errors.New(response.Status)
		}
	}

	message := strings.TrimSpace(r.Choices[0].Message.Content)
	left := strings.Index(message, `"""`)
	right := strings.LastIndex(message, `"""`)

	if left > -1 && left < right {
		message = strings.ReplaceAll(message[left+3:right], "\"", "")
		logrus.Infof("system assistant generate prompt[%s]: %s", model, message)
		return strings.TrimSpace(message), nil
	}

	if strings.HasSuffix(message, `"""`) { // 哎。bing 偶尔会漏掉前面的"""
		message = strings.ReplaceAll(message[:len(message)-3], "\"", "")
		logrus.Infof("system assistant generate prompt[%s]: %s", model, message)
		return strings.TrimSpace(message), nil
	}

	left = strings.Index(message, "```")
	right = strings.LastIndex(message, "```")

	if left > -1 && left < right {
		message = strings.ReplaceAll(message[left+3:right], "\"", "")
		logrus.Infof("system assistant generate prompt[%s]: %s", model, message)
		return strings.TrimSpace(message), nil
	}

	logrus.Info("response content: ", message)
	logrus.Errorf("system assistant generate prompt[%s] error: system assistant generate prompt failed", model)
	return "", errors.New("system assistant generate prompt failed")
}

func fetch(ctx context.Context, proxies, baseUrl, cookie string, marshal []byte) (*http.Response, error) {
	if strings.Contains(baseUrl, "127.0.0.1") || strings.Contains(baseUrl, "localhost") {
		proxies = ""
	}

	return common.ClientBuilder().
		Context(ctx).
		Proxies(proxies).
		POST(fmt.Sprintf("%s/v1/chat/completions", baseUrl)).
		Header("Authorization", cookie).
		JHeader().
		Bytes(marshal).
		Do()
}
