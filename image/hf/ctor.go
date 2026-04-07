package hf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/xllm-go/g"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
	"github.com/xllm-go/g/tokenizer"
)

const (
	baseURL   = "https://asahina2k-animagine-xl-4-0.hf.space"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0"
)

var (
	Sdk = g.Sdk()
)

func init() {
	Sdk.OnInitialized(func() {
		Sdk.Support("animagine-xl-4.0").
			Image(func(ctx *model.Ctx) (err error) {
				generation := ctx.GetGeneration()
				sessionHash := hash()
				chunk := fmt.Sprintf(`{"data":[],"event_data":null,"fn_index":4,"trigger_id":43,"session_hash":"%s"}`, sessionHash)
				request, err := http.NewRequest(http.MethodPost, "https://asahina2k-animagine-xl-4-0.hf.space/queue/join?__theme=light", strings.NewReader(chunk))
				if err != nil {
					return
				}

				request.Header.Set("content-type", "application/json")
				request.Header.Set("user-agent", userAgent)
				request.Header.Set("origin", baseURL)
				response, err := Do(http.DefaultClient.Do(request))(isStatus(http.StatusOK), isJson)
				if err != nil {
					return
				}
				_ = response.Body.Close()

				response, err = Do(http.DefaultClient.Get(baseURL+"/queue/data?session_hash="+sessionHash))(isStatus(http.StatusOK), isStream)
				if err != nil {
					return
				}

				scanner, err := newScanner(ctx.Context(), response)
				if err != nil {
					return
				}

				scanner.Event("process_completed", func(j JoinEvent) (_ interface{}) {
					logger.Sugar().Debug("process completed")
					return
				})

				if err = scanner.Do(); err != nil {
					return
				}

				rmbg := false // 是否删除背景
				parser := tokenizer.New("tag")
				var elems []tokenizer.Elem
				for _, elem := range parser.Parse(generation.Message) {
					if elem.Kind() == tokenizer.Ident {
						switch elem.Expr() {
						case "tag": // 特殊标签
							if r, ok := elem.Boolean("rmbg"); ok {
								rmbg = r
							}
							continue
						}
					}
					elems = append(elems, elem)
				}
				generation.Message = tokenizer.Join(elems)

				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				data := map[string]interface{}{
					"data": []interface{}{
						generation.Message,
						"lowres, bad anatomy, bad hands, text, error, missing finger, extra digits, fewer digits, cropped, worst quality, low quality, low score, bad score, average score, signature, watermark, username, blurry",
						r.Intn(79367128) + 800000000,
						1024,
						1024,
						5,
						28,
						generation.Quality,
						"1024 x 1024",
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

				request, err = http.NewRequest(http.MethodPost, "https://asahina2k-animagine-xl-4-0.hf.space/queue/join?__theme=light", bytes.NewReader(buf))
				if err != nil {
					return
				}

				request.Header.Set("content-type", "application/json")
				request.Header.Set("user-agent", userAgent)
				request.Header.Set("origin", baseURL)
				response, err = Do(http.DefaultClient.Do(request))(isStatus(http.StatusOK), isJson)
				if err != nil {
					return
				}

				response, err = Do(http.DefaultClient.Get(baseURL+"/queue/data?session_hash="+sessionHash))(isStatus(http.StatusOK), isStream)
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
						scanner.Failed(fmt.Errorf("process completed but not success: %s", j.InitialBytes))
						return
					}

					if len(j.Output.Data) == 0 {
						scanner.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
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
					"prompt": generation.Message,
					"data": []map[string]string{
						{
							"url": value,
						},
					},
				})
			})
	})
}
