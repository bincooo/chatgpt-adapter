package hf

import (
	"bypass/tool"
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xllm-go/g"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0"
)

var (
	Sdk = g.Sdk()
)

func init() {
	Sdk.OnInitialized(func() {
		Sdk.Support("animagine-xl-4.0").
			Image(func(ctx *model.Ctx) (err error) {
				baseUrl := "https://asahina2k-animagine-xl-4-0.hf.space"
				generation := ctx.GetGeneration()
				sessionHash := hash()
				chunk := fmt.Sprintf(`{"data":[],"event_data":null,"fn_index":4,"trigger_id":43,"session_hash":"%s"}`, sessionHash)
				request, err := http.NewRequest(http.MethodPost, baseUrl+"/queue/join?__theme=light", strings.NewReader(chunk))
				if err != nil {
					return
				}

				request.Header.Set("content-type", "application/json")
				request.Header.Set("user-agent", userAgent)
				request.Header.Set("origin", baseUrl)
				response, err := Do(http.DefaultClient.Do(request))(isStatus(http.StatusOK), isJson)
				if err != nil {
					return
				}
				_ = response.Body.Close()

				response, err = Do(http.DefaultClient.Get(baseUrl+"/queue/data?session_hash="+sessionHash))(isStatus(http.StatusOK), isStream)
				if err != nil {
					return
				}

				scanner, err := newScanner(ctx.Context(), response)
				if err != nil {
					return
				}
				if err = scanner.Do(); err != nil {
					return
				}

				var (
					w        = 1024
					h        = 1024
					scale    = 7.2
					steps    = 30
					negative = "lowres, bad anatomy, bad hands, text, error, missing finger, extra digits, fewer digits, cropped, worst quality, low quality, low score, bad score, average score, signature, watermark, username, blurry"

					rmbg = false // 是否删除背景
				)

				if generation.Size != "" {
					reg := regexp.MustCompile(`\s*x\s*`)
					x := reg.Split(generation.Size, -1)
					if len(x) == 2 {
						w, err = strconv.Atoi(strings.TrimSpace(x[0]))
						if err != nil {
							return
						}
						h, err = strconv.Atoi(strings.TrimSpace(x[1]))
						if err != nil {
							return
						}
					}
				}

				if generation.Extra != nil {
					if generation.Extra.Contains("scale") {
						scale = generation.Extra.Get("scale").(float64)
					}
					if generation.Extra.Contains("steps") {
						steps = int(generation.Extra.Get("steps").(float64))
					}
					if generation.Extra.Contains("negative") {
						negative = generation.Extra.Get("negative").(string)
					}
					if generation.Extra.Contains("rmbg") {
						rmbg = generation.Extra.Get("rmbg").(bool)
					}
				}

				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				data := map[string]interface{}{
					"data": []interface{}{
						generation.Message,
						negative,
						r.Intn(2047483647) + 80000000,
						w,
						h,
						scale,
						steps,
						generation.Quality,
						fmt.Sprintf("%d x %d", w, h),
						generation.Style,
						true,
						0.55,
						1.5,
						true,
					},
					"event_data":   nil,
					"fn_index":     5,
					"trigger_id":   43,
					"session_hash": sessionHash,
				}
				buf, err := json.Marshal(data)
				if err != nil {
					return
				}

				request, err = http.NewRequest(http.MethodPost, baseUrl+"/queue/join?__theme=light", bytes.NewReader(buf))
				if err != nil {
					return
				}

				request.Header.Set("content-type", "application/json")
				request.Header.Set("user-agent", userAgent)
				request.Header.Set("origin", baseUrl)
				response, err = Do(http.DefaultClient.Do(request))(isStatus(http.StatusOK), isJson)
				if err != nil {
					return
				}

				response, err = Do(http.DefaultClient.Get(baseUrl+"/queue/data?session_hash="+sessionHash))(isStatus(http.StatusOK), isStream)
				if err != nil {
					return
				}

				scanner, err = newScanner(ctx.Context(), response)
				if err != nil {
					return
				}

				value := ""
				scanner.Event("process_completed", func(j JoinEvent) (_ interface{}) {
					logger.Sugar().Debug("process completed")
					if !j.Success {
						scanner.Failed(fmt.Errorf("process completed but not success: %s", j.Output.Err))
						return
					}

					if len(j.Output.Data) == 0 {
						scanner.Failed(fmt.Errorf("image generate failed: %s", j.Output.Err))
						return
					}

					i := j.Output.Data[0].([]interface{})
					dict := i[0].(map[string]interface{})
					info := dict["image"].(map[string]interface{})
					value = info["url"].(string)
					return
				})

				if err = scanner.Do(); err != nil {
					return
				}

				// 执行背景删除
				if rmbg {
					value, err = removeBackground(ctx, value)
					if err != nil {
						return
					}
				} else {
					buf, err = tool.Download(value, map[string]string{
						"origin":     "https://huggingface.co",
						"referer":    baseUrl + "/?__theme=light",
						"user-agent": userAgent,
					})
					if err != nil {
						return
					}

					value, _, err = tool.Upload(buf, "png")
					if err != nil {
						return
					}
					//value = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf)
				}

				return ctx.Writer(model.Record[string, interface{}]{
					"created": time.Now().Unix(),
					"samples": []string{
						"DPM++ 2M Karras",
						"DPM++ SDE Karras",
						"DPM++ 2M SDE Karras",
						"Euler",
						"Euler a",
						"DDIM",
					},
					"styles": []string{
						"(None)",
						"Anim4gine",
						"Painting",
						"Pixel art",
						"1980s",
						"1990s",
						"2000s",
						"Toon",
						"Lineart",
						"Art Nouveau",
						"Western Comics",
						"3D",
						"Realistic",
						"Neonpunk",
					},
					"data": []map[string]string{
						{
							"url": value,
						},
					},
				})
			}, 2, 3)

		Sdk.Support("z-image-turbo").
			Image(func(ctx *model.Ctx) (err error) {
				baseUrl := "https://prithivmlmods-z-image-turbo-lora-dlc.hf.space"
				generation := ctx.GetGeneration()
				sessionHash := hash()

				var (
					w     = 1024
					h     = 1024
					steps = 9
					scale = 20

					rmbg = false // 是否删除背景
				)

				if generation.Size != "" {
					reg := regexp.MustCompile(`\s*x\s*`)
					x := reg.Split(generation.Size, -1)
					if len(x) == 2 {
						w, err = strconv.Atoi(strings.TrimSpace(x[0]))
						if err != nil {
							return
						}
						h, err = strconv.Atoi(strings.TrimSpace(x[1]))
						if err != nil {
							return
						}
					}
				}

				if generation.Extra != nil {
					if generation.Extra.Contains("steps") {
						steps = int(generation.Extra.Get("steps").(float64))
					}
					if generation.Extra.Contains("scale") {
						scale = int(generation.Extra.Get("scale").(float64))
					}
					if generation.Extra.Contains("rmbg") {
						rmbg = generation.Extra.Get("rmbg").(bool)
					}
				}

				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				data := map[string]interface{}{
					"data": []interface{}{
						generation.Message,
						nil,
						0.75,
						scale,
						steps,
						nil,
						true,
						r.Intn(2047483647) + 80000000,
						w,
						h,
						0.95,
					},
					"fn_index":     3,
					"trigger_id":   8,
					"session_hash": sessionHash,
				}
				buf, err := json.Marshal(data)
				if err != nil {
					return
				}

				request, err := http.NewRequest(http.MethodPost, baseUrl+"/gradio_api/queue/join?__theme=light", bytes.NewReader(buf))
				if err != nil {
					return
				}

				request.Header.Set("content-type", "application/json")
				request.Header.Set("user-agent", userAgent)
				request.Header.Set("origin", baseUrl)
				response, err := Do(http.DefaultClient.Do(request))(isStatus(http.StatusOK), isJson)
				if err != nil {
					return
				}
				_ = response.Body.Close()

				response, err = Do(http.DefaultClient.Get(baseUrl+"/gradio_api/queue/data?session_hash="+sessionHash))(isStatus(http.StatusOK), isStream)
				if err != nil {
					return
				}

				scanner, err := newScanner(ctx.Context(), response)
				if err != nil {
					return
				}

				value := ""
				scanner.Event("process_completed", func(j JoinEvent) (_ interface{}) {
					logger.Sugar().Debug("process completed")
					if !j.Success {
						scanner.Failed(fmt.Errorf("process completed but not success: %s", j.Output.Err))
						return
					}

					if len(j.Output.Data) == 0 {
						scanner.Failed(fmt.Errorf("image generate failed: %s", j.Output.Err))
						return
					}

					dict := j.Output.Data[0].(map[string]interface{})
					value = dict["url"].(string)
					return
				})

				if err = scanner.Do(); err != nil {
					return
				}

				// 执行背景删除
				if rmbg {
					value, err = removeBackground(ctx, value)
					if err != nil {
						return
					}
				} else {
					buf, err = tool.Download(value, map[string]string{
						"origin":     "https://huggingface.co",
						"referer":    baseUrl + "/?__theme=light",
						"user-agent": userAgent,
					})
					if err != nil {
						return
					}

					value, _, err = tool.Upload(buf, "png")
					if err != nil {
						return
					}
					//value = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf)
				}

				return ctx.Writer(model.Record[string, interface{}]{
					"created": time.Now().Unix(),
					"data": []map[string]string{
						{
							"url": value,
						},
					},
				})
			}, 2, 3)
	})
}
