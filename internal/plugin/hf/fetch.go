package hf

import (
	"fmt"
	com "github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const negative = "(deformed eyes, nose, ears, nose, leg, head), bad anatomy, ugly"

func xl(ctx *gin.Context, model, samples, message string) (value string, err error) {
	var (
		index   = 0
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		baseUrl = "wss://prodia-sdxl-stable-diffusion-xl.hf.space"
		domain  = pkg.Config.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	conn, err := emit.SocketBuilder().
		Proxies(proxies).
		URL(baseUrl + "/queue/join").
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		return
	}

	c, err := emit.NewGio(ctx.Request.Context(), conn)
	if err != nil {
		return
	}

	c.Event("send_hash", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     index,
			"session_hash": hash,
		}
	})

	c.Event("send_data", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     index,
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

		if file, err = com.SaveBase64(d[0].(string), "png"); err != nil {
			c.Failed(fmt.Errorf("image save failed: %s", j.InitialBytes))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, file)
		return
	})

	err = c.Do()
	return
}

func sd(ctx *gin.Context, model, samples, message string) (value string, err error) {
	var (
		index   = 0
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		baseUrl = "https://prodia-fast-stable-diffusion.hf.space"
		domain  = pkg.Config.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	response, err := emit.ClientBuilder().
		Context(ctx.Request.Context()).
		Proxies(proxies).
		GET(baseUrl+"/queue/join").
		Query("fn_index", strconv.Itoa(index)).
		Query("session_hash", hash).
		DoS(http.StatusOK)
	if err != nil {
		return
	}

	c, err := emit.NewGio(ctx.Request.Context(), response)
	if err != nil {
		return
	}

	c.Event("send_data", func(j emit.JoinEvent) (_ interface{}) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		_, err = emit.ClientBuilder().
			Proxies(proxies).
			Context(ctx.Request.Context()).
			POST(baseUrl + "/queue/data").
			JHeader().
			Body(map[string]interface{}{
				"fn_index":     index,
				"session_hash": hash,
				"event_id":     j.EventId,
				"trigger_id":   r.Intn(15) + 5,
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
			}).
			DoS(http.StatusOK)
		if err != nil {
			c.Failed(err)
		}
		return
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data
		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		result := d[0].(map[string]interface{})
		value, err = com.Download(proxies, fmt.Sprintf("%s/file=%s", baseUrl, result["path"].(string)), "png")
		if err != nil {
			c.Failed(err)
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, value)
		return
	})

	err = c.Do()
	return
}

func google(ctx *gin.Context, model, message string) (value string, err error) {
	var (
		index   = 3
		hash    = emit.GioHash()
		proxies = ctx.GetString("proxies")
		baseUrl = "wss://google-sdxl.hf.space"
		domain  = pkg.Config.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	conn, err := emit.SocketBuilder().
		Proxies(proxies).
		URL(baseUrl + "/queue/join").
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		return
	}

	c, err := emit.NewGio(ctx.Request.Context(), conn)
	if err != nil {
		return
	}

	c.Event("send_hash", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     index,
			"session_hash": hash,
		}
	})

	c.Event("send_data", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     index,
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
		if file, err = com.SaveBase64(values[r.Intn(len(values))].(string), "jpg"); err != nil {
			c.Failed(fmt.Errorf("image save failed: %s", j.InitialBytes))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, file)
		return
	})

	err = c.Do()
	return
}
