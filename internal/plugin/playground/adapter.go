package pg

import (
	"bytes"
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"encoding/json"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type modelPayload struct {
	BatchId                 string  `json:"batchId"`
	CfgScale                int32   `json:"cfg_scale"`
	BoothModel              string  `json:"dream_booth_model"`
	Filter                  string  `json:"filter"`
	GenerateVariants        bool    `json:"generateVariants"`
	GuidanceScale           int32   `json:"guidance_scale"`
	Width                   int32   `json:"width"`
	Height                  int32   `json:"height"`
	HighNoiseFrac           float32 `json:"high_noise_frac"`
	InitImageFromPlayground bool    `json:"initImageFromPlayground"`
	IsPrivate               bool    `json:"isPrivate"`
	ModelType               string  `json:"modelType"`
	NegativePrompt          string  `json:"negativePrompt"`
	NumImages               int32   `json:"num_images"`
	Prompt                  string  `json:"prompt"`
	Sampler                 int32   `json:"sampler"`
	Seed                    int32   `json:"seed"`
	StatusUUID              string  `json:"statusUUID"`
	Steps                   int32   `json:"steps"`
	Strength                float32 `json:"strength"`
}

type modelCompleted struct {
	Meta struct {
		NumImagesInLast24Hours int32 `json:"numImagesInLast24Hours"`
	} `json:"meta"`
	Images []struct {
		ImageKey string `json:"imageKey"`
		Prompt   string `json:"prompt"`
		Url      string `json:"url"`
		Loading  bool   `json:"loading"`
	} `json:"images"`
}

var (
	Adapter = API{}
	models  = []string{
		"none",
		"Realism_Engine_SDXL",
		"Real_Cartoon_XL",
		"Blue_Pencil_XL",
		"Starlight_XL",
		"Juggernaut_XL",
		"RealVisXL",
		"ZavyChromaXL",
		"NightVision_XL",
		"Realistic_Stock_Photo",
		"DreamShaper",
		"MBBXL_Ultimate",
		"Mysterious",
		"Copax_TimeLessXL",
		"SDXL_Niji",
		"Pixel_Art_XL",
		"ProtoVision_XL",
		"DucHaiten_AIart_SDXL",
		"CounterfeitXL",
		"vibrant_glass",
		"dreamy_stickers",
		"ultra_lighting",
		"watercolor",
		"macro_realism",
		"delicate_detail",
		"radiant_symmetry",
		"lush_illustration",
		"saturated_space",
		"neon_mecha",
		"ethereal_low_poly",
		"warm_box",
		"cinematic",
		"cinematic_warm",
		"wasteland",
		"flat_palette",
		"ominous_escape",
		"spielberg",
		"royalistic",
		"masterpiece",
		"wall_art",
		"haze",
		"black_and_white_3d",
	}
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(ctx *gin.Context, model string) (ok bool) {
	token := ctx.GetString("token")
	if model == "dall-e-3" {
		ok, _ = regexp.MatchString(`\w{8,10}-\w{4}-\w{4}-\w{4}-\w{10,15}`, token)
	}
	return
}

func (API) Generation(ctx *gin.Context) {

	var (
		hash       = emit.GioHash()
		cookie     = ctx.GetString("token")
		domain     = pkg.Config.GetString("domain")
		generation = common.GetGinGeneration(ctx)
	)

	model := matchModel(generation.Style)
	var payload = modelPayload{
		BatchId:                 hash,
		CfgScale:                8,
		GuidanceScale:           8,
		Width:                   1024,
		Height:                  1024,
		HighNoiseFrac:           0.8,
		GenerateVariants:        false,
		InitImageFromPlayground: false,
		IsPrivate:               false,
		ModelType:               "stable-diffusion-xl",
		Filter:                  model,
		BoothModel:              model,
		NegativePrompt:          "ugly, deformed, noisy, blurry, distorted, out of focus, bad anatomy, extra limbs, poorly drawn face, poorly drawn hands, missing fingers, ugly, deformed, noisy, blurry, distorted, out of focus, bad anatomy, extra limbs, poorly drawn face, poorly drawn hands, missing fingers, photo, realistic, text, watermark, signature, username, artist name",
		NumImages:               1,
		Prompt:                  generation.Message,
		Sampler:                 9,
		Seed:                    int32(rand.Intn(100000000) + 429650152),
		StatusUUID:              uuid.NewString(),
		Steps:                   30,
		Strength:                1.45,
	}

	marshal, _ := json.Marshal(payload)
	r, err := fetch(common.GetGinContext(ctx), "", cookie, marshal)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	// {"errorCode":
	if bytes.HasPrefix(data, []byte("{\"errorCode\":")) {
		logger.Error(err)
		response.Error(ctx, -1, string(data))
		return
	}

	var mc modelCompleted
	if err = json.Unmarshal(data, &mc); err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if len(mc.Images) == 0 {
		response.Error(ctx, -1, "generate images failed")
		return
	}

	file, err := common.SaveBase64(mc.Images[0].Url, "jpg")
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}
	file = fmt.Sprintf("%s/file/%s", domain, file)
	if (generation.Size == "HD" || strings.HasPrefix(generation.Size, "1792x")) && common.HasMfy() {
		v, e := common.Magnify(ctx, file)
		if e != nil {
			logger.Error(e)
		} else {
			file = v
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"styles":  models,
		"data": []map[string]string{
			{"url": file},
		},
		"currStyle": model,
	})
}

func matchModel(style string) string {
	if common.Contains(models, style) {
		return style
	}
	return models[rand.Intn(len(models))]
}

func fetch(ctx context.Context, proxies, cookie string, marshal []byte) (*http.Response, error) {
	if !strings.Contains(cookie, "__Secure-next-auth.session-token=") {
		cookie = "__Secure-next-auth.session-token=" + cookie
	}

	baseUrl := "https://playground.com"
	return emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(ctx).
		POST(baseUrl+"/api/models").
		Header("host", "playground.com").
		Header("origin", "https://playground.com").
		Header("referer", "https://playground.com/create").
		Header("accept-language", "en-US,en;q=0.9").
		Header("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36").
		Header("x-forwarded-for", emit.RandIP()).
		Header("cookie", cookie).
		JHeader().
		Bytes(marshal).
		DoS(http.StatusOK)
}
