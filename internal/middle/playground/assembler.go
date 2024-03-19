package pg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/bincooo/sdio"
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

func Generation(ctx *gin.Context, req gpt.ChatGenerationRequest) {
	hash := sdio.SessionHash()
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
		domain  = pkg.Config.GetString("domain")
	)

	model := convertToModel(req.Style)
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
		Prompt:                  req.Prompt,
		Sampler:                 9,
		Seed:                    int32(rand.Intn(100000000) + 429650152),
		StatusUUID:              uuid.NewString(),
		Steps:                   30,
		Strength:                1.45,
	}

	marshal, _ := json.Marshal(payload)
	response, err := fetch(ctx.Request.Context(), "", cookie, marshal)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if err = middle.IsCanceled(ctx); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	// {"errorCode":
	if bytes.HasPrefix(data, []byte("{\"errorCode\":")) {
		middle.ResponseWithV(ctx, -1, string(data))
		return
	}

	var mc modelCompleted
	if err = json.Unmarshal(data, &mc); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if len(mc.Images) == 0 {
		middle.ResponseWithV(ctx, -1, "generate images failed")
		return
	}

	if err = middle.IsCanceled(ctx); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}
	file, err := common.CreateBase64Image(mc.Images[0].Url, "jpg")
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if domain == "" {
		file = fmt.Sprintf("http://127.0.0.1:%d/file/%s", ctx.GetInt("port"), file)
		ctx.JSON(http.StatusOK, gin.H{
			"created": time.Now().Unix(),
			"styles":  models,
			"data": []map[string]string{
				{"url": file},
			},
			"currStyle": model,
		})
		return
	}

	if err = middle.IsCanceled(ctx); err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}
	file, err = common.UploadCatboxFile(proxies, domain+"/file/"+file)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
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

func convertToModel(style string) string {
	if common.Contains(models, style) {
		return style
	}
	return models[rand.Intn(len(models))]
}

func fetch(ctx context.Context, proxies, cookie string, marshal []byte) (*http.Response, error) {
	if !strings.Contains(cookie, "__Secure-next-auth.session-token=") {
		cookie = "__Secure-next-auth.session-token=" + cookie
	}

	client, err := common.NewHttpClient(proxies)
	if err != nil {
		return nil, err
	}

	baseUrl := "https://playground.com"
	request, err := http.NewRequest(http.MethodPost, baseUrl+"/api/models", bytes.NewReader(marshal))
	if err != nil {
		return nil, err
	}

	h := request.Header
	h.Add("host", "playground.com")
	h.Add("origin", "https://playground.com")
	h.Add("referer", "https://playground.com/create")
	h.Add("accept-language", "en-US,en;q=0.9")
	h.Add("content-type", "application/json")
	h.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	h.Add("x-forwarded-for", common.RandomIp())
	h.Add("cookie", cookie)

	if err = middle.IsCanceled(ctx); err != nil {
		return nil, err
	}

	response, e := client.Do(request.WithContext(ctx))
	if e != nil {
		return nil, e
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.New(response.Status)
	}
	return response, nil

}
