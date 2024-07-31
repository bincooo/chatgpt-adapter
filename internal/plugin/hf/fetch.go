package hf

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	negative  = "(deformed eyes, nose, ears, nose, leg, head), bad anatomy, ugly"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0"
)

func Ox000(ctx *gin.Context, model, samples, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		baseUrl = "https://prodia-fast-stable-diffusion.hf.space"
		domain  = pkg.Config.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	fn := []int{0, 15}
	data := []interface{}{
		message + ", {{{{by famous artist}}}, beautiful, 4k",
		negative,
		model,
		25,
		samples,
		10,
		1024,
		1024,
		-1,
	}
	response, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(baseUrl+"/queue/join").
		JHeader().
		Header("User-Agent", userAgent).
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}

	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", userAgent).
		DoS(http.StatusOK)
	if err != nil {
		return
	}

	defer response.Body.Close()
	c, err := emit.NewGio(common.GetGinContext(ctx), response)
	if err != nil {
		return
	}

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data
		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}
		result := d[0].(map[string]interface{})
		value = result["url"].(string)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func Ox001(ctx *gin.Context, model, samples, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		baseUrl = "wss://prodia-sdxl-stable-diffusion-xl.hf.space"
		domain  = pkg.Config.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	conn, response, err := emit.SocketBuilder(plugin.HTTPClient).
		Proxies(proxies).
		URL(baseUrl + "/queue/join").
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		return
	}

	defer response.Body.Close()
	c, err := emit.NewGio(common.GetGinContext(ctx), conn)
	if err != nil {
		return
	}

	c.Event("send_hash", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"session_hash": hash,
		}
	})

	c.Event("send_data", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"session_hash": hash,
			"data": []interface{}{
				message + ", {{{{by famous artist}}}, beautiful, 4k",
				negative,
				model,
				25,
				samples,
				10,
				1024,
				1024,
				-1,
			},
		}
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		var file string
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		if file, err = common.SaveBase64(d[0].(string), "png"); err != nil {
			c.Failed(fmt.Errorf("image save failed: %s", j.InitialBytes))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, file)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func Ox002(ctx *gin.Context, model, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		baseUrl = "https://mukaist-dalle-4k.hf.space"
	)

	if u := pkg.Config.GetString("hf.dalle-4k.baseUrl"); u != "" {
		baseUrl = u
	}

	fn := []int{3, 6}
	data := []interface{}{
		message,
		negative,
		true,
		model,
		30,
		1024,
		1024,
		6,
		true,
	}
	fn, data, err = bindAttr("dalle-4k", fn, data, message, negative, "", model, -1)
	response, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(baseUrl+"/queue/join").
		JHeader().
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}

	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()
	response, err = emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		return
	}

	defer response.Body.Close()
	c, err := emit.NewGio(common.GetGinContext(ctx), response)
	if err != nil {
		return
	}

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}
		if len(values) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		v, ok := values[0].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		value = v["image"].(map[string]interface{})["url"].(string)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func Ox003(ctx *gin.Context, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		domain  = pkg.Config.GetString("domain")
		baseUrl = "https://ehristoforu-dalle-3-xl-lora-v2.hf.space"
		r       = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	if u := pkg.Config.GetString("hf.dalle-3-xl.baseUrl"); u != "" {
		baseUrl = u
	}

	fn := []int{3, 6}
	data := []interface{}{
		message + ", {{{{by famous artist}}}, beautiful, 4k",
		negative + ", extra limb, missing limb, floating limbs, (mutated hands and fingers:1.4), disconnected limbs, mutation, mutated, ugly, disgusting, blurry, amputation",
		true,
		r.Intn(51206501) + 1100000000,
		1024,
		1024,
		12,
		true,
	}
	fn, data, err = bindAttr("dalle-3-xl", fn, data, message, negative, "", "", r.Intn(51206501)+1100000000)
	response, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(baseUrl+"/queue/join").
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept-Language", "en-US,en;q=0.9").
		JHeader().
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return "", err
	}

	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept", "text/event-stream").
		Header("Accept-Language", "en-US,en;q=0.9").
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	c, err := emit.NewGio(ctx.Request.Context(), response)
	if err != nil {
		return "", err
	}

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		if len(values) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		v, ok := values[0].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		info, ok := v["image"].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		// 锁环境了，只能先下载下来
		value, err = common.Download(plugin.HTTPClient, proxies, info["url"].(string), "png", map[string]string{
			//"User-Agent":      userAgent,
			//"Accept-Language": "en-US,en;q=0.9",
			"Origin":  "https://huggingface.co",
			"Referer": baseUrl + "/?__theme=light",
		})
		if err != nil {
			c.Failed(fmt.Errorf("image download failed: %v", err))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, value)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

// 潦草漫画的风格
func Ox004(ctx *gin.Context, model, samples, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		domain  = pkg.Config.GetString("domain")
		baseUrl = "https://cagliostrolab-animagine-xl-3-1.hf.space"
		r       = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	if u := pkg.Config.GetString("hf.animagine-xl-3.1.baseUrl"); u != "" {
		baseUrl = u
	}

	fn := []int{5, 49}
	data := []interface{}{
		message,
		"(text:1.3), (strip cartoon:1.3), worst quality, low quality",
		r.Intn(1490935504) + 9068457,
		1024,
		1024,
		7,
		35,
		samples,
		"1024 x 1024",
		model,
		"Heavy v3.1",
		false,
		0.55,
		1.5,
		true,
	}
	fn, data, err = bindAttr("animagine-xl-3.1", fn, data, message, negative, samples, model, r.Intn(1490935504)+9068457)
	response, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(baseUrl+"/queue/join").
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept-Language", "en-US,en;q=0.9").
		JHeader().
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return "", err
	}
	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept", "text/event-stream").
		Header("Accept-Language", "en-US,en;q=0.9").
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	c, err := emit.NewGio(common.GetGinContext(ctx), response)
	if err != nil {
		return "", err
	}

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		if len(values) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		v, ok := values[0].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		info, ok := v["image"].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		// 锁环境了，只能先下载下来
		value, err = common.Download(plugin.HTTPClient, proxies, info["url"].(string), "png", map[string]string{
			//"User-Agent":      userAgent,
			//"Accept-Language": "en-US,en;q=0.9",
			"Origin":  "https://huggingface.co",
			"Referer": baseUrl + "/?__theme=light",
		})
		if err != nil {
			c.Failed(fmt.Errorf("image download failed: %v", err))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, value)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func google(ctx *gin.Context, model, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		baseUrl = "wss://google-sdxl.hf.space"
		domain  = pkg.Config.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	conn, response, err := emit.SocketBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		URL(baseUrl + "/queue/join").
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		return
	}
	defer response.Body.Close()

	c, err := emit.NewGio(common.GetGinContext(ctx), conn)
	if err != nil {
		return
	}

	c.Event("send_hash", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     2,
			"session_hash": hash,
		}
	})

	c.Event("send_data", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     2,
			"session_hash": hash,
			"data": []interface{}{
				message + ", {{{{by famous artist}}}, beautiful, 4k",
				negative,
				25,
				model,
			},
		}
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		var file string
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if file, err = common.SaveBase64(values[r.Intn(len(values))].(string), "jpg"); err != nil {
			c.Failed(fmt.Errorf("image save failed: %s", j.InitialBytes))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, file)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func bindAttr(key string, fn []int, data []interface{}, message, negative, sampler, style string, seed int) ([]int, []interface{}, error) {
	slice := pkg.Config.GetIntSlice("hf." + key + ".fn")
	if len(slice) >= 2 {
		fn = slice
	}
	dataStr := pkg.Config.GetString("hf." + key + ".data")
	if dataStr != "" {
		dataStr = strings.ReplaceAll(dataStr, "{{prompt}}", message)
		dataStr = strings.ReplaceAll(dataStr, "{{negative_prompt}}", negative)
		dataStr = strings.ReplaceAll(dataStr, "{{sampler}}", sampler)
		dataStr = strings.ReplaceAll(dataStr, "{{style}}", style)
		dataStr = strings.ReplaceAll(dataStr, "{{seed}}", strconv.Itoa(seed))
		err := json.Unmarshal([]byte(dataStr), &data)
		if err != nil {
			return nil, nil, err
		}
	}
	return fn, data, nil
}
