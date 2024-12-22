package pg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iocgo/sdk/env"
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
	models = []string{
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

type pg struct {
	inter.BaseAdapter
	env *env.Environment
}

func (p *pg) Match(ctx *gin.Context, model string) (ok bool, err error) {
	token := ctx.GetString("token")
	if model == "dall-e-3" {
		ok, _ = regexp.MatchString(`\w{8,10}-\w{4}-\w{4}-\w{4}-\w{10,15}`, token)
	}
	return
}

func (p *pg) Generation(ctx *gin.Context) (err error) {

	var (
		hash       = emit.GioHash()
		cookie     = ctx.GetString("token")
		generation = common.GetGinGeneration(ctx)
	)

	mod := matchModel(generation.Style)
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
		Filter:                  mod,
		BoothModel:              mod,
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
	r, err := fetch(ctx, "", cookie, marshal)
	if err != nil {
		logger.Error(err)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		return
	}

	// {"errorCode":
	if bytes.HasPrefix(data, []byte("{\"errorCode\":")) {
		logger.Error(err)
		return
	}

	var mc modelCompleted
	if err = json.Unmarshal(data, &mc); err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if len(mc.Images) == 0 {
		err = errors.New("generate images failed")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"styles":  models,
		"data": []map[string]string{
			{"url": mc.Images[0].Url},
		},
		"currStyle": mod,
	})
	return
}

func matchModel(style string) string {
	if slices.Contains(models, style) {
		return style
	}
	return models[rand.Intn(len(models))]
}

func fetch(ctx context.Context, proxies, cookie string, marshal []byte) (*http.Response, error) {
	if !strings.Contains(cookie, "__Secure-next-auth.session-token=") {
		cookie = "__Secure-next-auth.session-token=" + cookie
	}

	baseUrl := "https://playground.com"
	return emit.ClientBuilder(common.HTTPClient).
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
		JSONHeader().
		Bytes(marshal).
		DoS(http.StatusOK)
}
