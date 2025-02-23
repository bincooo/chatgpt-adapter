package you

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/proxy"
)

const iCookie = "_ga_2N7ZM9C56V=GS1.1.1734870573.1.0.1734870585.0.0.1930923381; ab.storage.userId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3AU2qZWTUI96mhUL2k8i7A2DGf8qgc%7Ce%3Aundefined%7Cc%3A1734870585296%7Cl%3A1734870585299; DSR=eyJhbGciOiJSUzI1NiIsImtpZCI6IlNLMmpJbnU3SWpjMkp1eFJad1psWHBZRUpQQkFvIiwidHlwIjoiSldUIn0.eyJhbXIiOlsiZW1haWwiXSwiYXV0aDBJZCI6bnVsbCwiZHJuIjoiRFNSIiwiZW1haWwiOiJrd2d5YjFmMzQwQHNteWt3Yi5jb20iLCJleHAiOjE3NjYzMjAxODUsImdpdmVuTmFtZSI6IiIsImlhdCI6MTczNDg3MDU4NSwiaXNzIjoiUDJqSW50dFJNdVhweVlaTWJWY3NjNEM5WjBSVCIsImxhc3ROYW1lIjoiIiwibmFtZSI6IiIsInBpY3R1cmUiOiIiLCJzdHl0Y2hJZCI6bnVsbCwic3ViIjoiVTJxWldUVUk5Nm1oVUwyazhpN0EyREdmOHFnYyIsInRlbmFudEludml0YXRpb24iOm51bGwsInRlbmFudEludml0ZXIiOm51bGwsInVzZXJJZCI6IlUycVpXVFVJOTZtaFVMMms4aTdBMkRHZjhxZ2MiLCJ2ZXJpZmllZEVtYWlsIjp0cnVlfQ.jGdMsttqIQRPaT0wMHP9o0dQVKemk5dAHSBr6huP5ovB-c42c8Gd7gM_hFcoLuozX0NLrwATG0sDSxLFWJ8JkxvswQvM0yWqRknRBLrC1ZJ2KGsg2UeleMv4ApSglMn1Q1PorW3z5WgzeD0sbB0fWGf22uodiVB-bGNuWGtO9iFEoMKRH_R-VH91cFR1nxM95rzyf4Qm1-augava4_MCYBoOpXzQS5KVbLWRtvLYQCZycwBbpy_-WavykpNzIT15mEEi1CEEDVCB4R3x0WzT-ngbcsHJ3DzVRbw0bpF9EQoBW62bIuRVlndNmdKYqaJMKazE8Srt5uNZoHk5d4czww; _gtmeec=eyJjb3VudHJ5IjoiNzgwMzI4NThiNTIwMDJlZTVkZTg2Nzk5ZjU3NjliY2NiNjk5YmQ2NjhkN2RlNTY2MDczZmVhM2IzMDhjODJmNiJ9; _clsk=dwpwmi%7C1734870574146%7C1%7C1%7Cn.clarity.ms%2Fcollect; _gcl_au=1.1.1810051233.1734870573; FPAU=1.1.1810051233.1734870573; daily_query_date=Sun%20Dec%2022%202024; _ga=GA1.1.1458246791.1734870573; ab.storage.deviceId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3A824a2ecd-66ed-9747-dc77-2c719a290641%7Ce%3Aundefined%7Cc%3A1734870572460%7Cl%3A1734870585298; _clck=3515nu%7C2%7Cfrx%7C0%7C1817; FPGSID=1.1734870573.1734870573.G-WYGVQX1R23.bPbE_KJJvpZnm4qc15MT1w; youpro_subscription=false; ab.storage.sessionId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3Aa1e873fd-d6a7-1953-ec35-a33602bad270%7Ce%3A1734872385305%7Cc%3A1734870585297%7Cl%3A1734870585305; FPID=FPID2.2.9D%2BJrGD2pOYikNUYro03CIa2A4YB0EiIIyNlSidU6WE%3D.1734870573; DS=eyJhbGciOiJSUzI1NiIsImtpZCI6IlNLMmpJbnU3SWpjMkp1eFJad1psWHBZRUpQQkFvIiwidHlwIjoiSldUIn0.eyJhbXIiOlsiZW1haWwiXSwiYXV0aDBJZCI6bnVsbCwiZHJuIjoiRFMiLCJlbWFpbCI6Imt3Z3liMWYzNDBAc215a3diLmNvbSIsImV4cCI6MTczNjA4MDE4NSwiZ2l2ZW5OYW1lIjoiIiwiaWF0IjoxNzM0ODcwNTg1LCJpc3MiOiJQMmpJbnR0Uk11WHB5WVpNYlZjc2M0QzlaMFJUIiwibGFzdE5hbWUiOiIiLCJuYW1lIjoiIiwicGljdHVyZSI6IiIsInJleHAiOiIyMDI1LTEyLTIxVDEyOjI5OjQ1WiIsInN0eXRjaElkIjpudWxsLCJzdWIiOiJVMnFaV1RVSTk2bWhVTDJrOGk3QTJER2Y4cWdjIiwidGVuYW50SW52aXRhdGlvbiI6bnVsbCwidGVuYW50SW52aXRlciI6bnVsbCwidXNlcklkIjoiVTJxWldUVUk5Nm1oVUwyazhpN0EyREdmOHFnYyIsInZlcmlmaWVkRW1haWwiOnRydWV9.HE662jCQ5EhwcHnunk4yFdZrf7jWbczFURy_Jq9ZhALWK0TB6WE0CTGz84GA_Xc0AkOTN6eMCSE9jHjAnYGQeBcYE_JtxkV0JnyVULClCammqBXghwhIClV_IYMTYRqKVzHla9VwneHv2IQdBCd9yrB48daqUcruphwZiMFKMggmGXuiOgop8SBRNbdrw-pEWYq88-j7xiTOSBCUEBbDmyUQmo5uSJ6xYaqG6Iz3MfEgsBBtSxN8BvMKHx8LX0FWX1H1l3NFbQZ_s3LnTKqzhd_3ZkRWlY7ZV1-ScsGjL3_HrkgWp3HYI7u_V7nqZvFSrR1TvbxELhpOBdVTZHIujw; youchat_smart_learn=true; youchat_personalization=true; you_subscription=free; ai_model=gpt_4o; FPLC=EYFwldf7wUW3lQwsc13yt5HkZIJZJHKjTrz1h8d84v%2B8RybGzrrcWxPePaq%2F2NYYDgPfhJHfPX5VcECV6As9Cz%2B%2BT%2B8Vjjnw0k2WYc7IdMzB9JXc1Yx2lVQETLUmbw%3D%3D; ld_context=%7B%22kind%22%3A%22user%22%2C%22key%22%3A%2219737287-c346-4618-8842-1e029ef4e109%22%2C%22email%22%3A%22UNKNOWN%22%2C%22userCreatedAt%22%3Anull%2C%22country%22%3A%22HK%22%2C%22userAgent%22%3A%22Mozilla%2F5.0%20(Macintosh%3B%20Intel%20Mac%20OS%20X%2010_15_7)%20AppleWebKit%2F537.36%20(KHTML%2C%20like%20Gecko)%20Chrome%2F125.0.0.0%20Safari%2F537.36%20Edg%2F125.0.0.0%22%2C%22secUserAgent%22%3A%22%5C%22Google%20Chrome%5C%22%3Bv%3D%5C%22131%5C%22%2C%20%5C%22Chromium%5C%22%3Bv%3D%5C%22131%5C%22%2C%20%5C%22Not_A%20Brand%5C%22%3Bv%3D%5C%2224%5C%22%22%7D; uuid_guest_backup=c59bd98b-dee1-4d32-ba3a-0b650b9afda8; daily_query_count=0; total_query_count=0; safesearch_guest=Moderate; uuid_guest=c59bd98b-dee1-4d32-ba3a-0b650b9afda8"

var (
	mu sync.Mutex

	lang      = "cn-ZN,cn;q=0.9"
	clearance = ""
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0"

	cookiesContainer *common.PollContainer[string]
)

func init() {
	inited.AddInitialized(func(env *env.Environment) {
		cookies := env.GetStringSlice("you.cookies")
		cookiesContainer = common.NewPollContainer[string]("you", cookies, 6*time.Hour)
		cookiesContainer.Condition = condition(env)
		if len(cookies) > 0 && env.GetBool("you.task") {
			go timer(env)
		}
	})
}

func timer(env *env.Environment) {
	m30 := 30 * time.Minute

	for {
		time.Sleep(m30)
		if clearance != "" {
			timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			chat := you.New(iCookie, you.GPT_4, env.GetString("server.proxied"))
			chat.CloudFlare(clearance, userAgent, lang)
			chat.Client(common.HTTPClient)
			_, err := chat.State(timeout)
			cancel()
			if err == nil {
				continue
			}

			var se emit.Error
			if !errors.As(err, &se) {
				logger.Errorf("定时器 you.com 过盾检查失败：%v", err)
				continue
			}

			if se.Code == 403 {
				// 需要重新过盾
				clearance = ""
			} else {
				logger.Info("定时器执行 you.com 过盾检查，无需执行")
				continue
			}
		}

		// 尝试过盾
		if err := hookCloudflare(env); err != nil {
			logger.Errorf("you.com 尝试过盾失败：%v", err)
			continue
		}

		logger.Info("定时器执行 you.com 过盾成功")
	}
}

func InvocationHandler(ctx *proxy.Context) {
	var (
		gtx  = ctx.In[0].(*gin.Context)
		echo = gtx.GetBool(vars.GinEcho)
	)

	if echo || ctx.Method != "Completion" && ctx.Method != "ToolChoice" {
		ctx.Do()
		return
	}

	logger.Infof("execute static proxy [relay/llm/you.api]: func %s(...)", ctx.Method)

	if cookiesContainer.Len() == 0 {
		response.Error(gtx, -1, "empty cookies")
		return
	}

	cookies, err := cookiesContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(gtx, -1, err)
		return
	}
	defer resetMarked(cookies)
	gtx.Set("token", cookies)
	gtx.Set("clearance", clearance)
	gtx.Set("userAgent", userAgent)
	gtx.Set("lang", lang)

	//
	ctx.Do()

	//
	if ctx.Method == "Completion" {
		err = elseOf[error](ctx.Out[0])
	}
	if ctx.Method == "ToolChoice" {
		err = elseOf[error](ctx.Out[1])
	}

	if err != nil {
		logger.Error(err)
		var se emit.Error
		if errors.As(err, &se) && se.Code > 400 {
			_ = cookiesContainer.MarkTo(cookies, 2)
			// 403 重定向？？？
			if se.Code == 403 {
				cleanCloudflare()
			}
		}

		if strings.Contains(err.Error(), "ZERO QUOTA") {
			_ = cookiesContainer.MarkTo(cookies, 2)
		}
		return
	}
}

func condition(env *env.Environment) func(string, ...interface{}) bool {
	return func(cookies string, argv ...interface{}) bool {

		marker, err := cookiesContainer.Marked(cookies)
		if err != nil {
			logger.Error(err)
			return false
		}

		if marker != 0 {
			return false
		}

		// return true
		chat := you.New(cookies, you.CLAUDE_2, env.GetString("server.proxied"))
		chat.Client(common.HTTPClient)
		chat.CloudFlare(clearance, userAgent, lang)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// 检查可用次数
		count, err := chat.State(ctx)
		if err != nil {
			var se emit.Error
			if errors.As(err, &se) {
				if se.Code == 403 {
					cleanCloudflare()
					_ = hookCloudflare(env)
				}
				if se.Code == 401 { // cookie 失效？？？
					_ = cookiesContainer.MarkTo(cookies, 2)
				}
			}
			logger.Error(err)
			return false
		}

		if count <= 0 {
			_ = cookiesContainer.MarkTo(cookies, 2)
			return false
		}

		return true
	}
}

func resetMarked(cookies string) {
	marker, err := cookiesContainer.Marked(cookies)
	if err != nil {
		logger.Error(err)
		return
	}

	if marker != 1 {
		return
	}

	err = cookiesContainer.MarkTo(cookies, 0)
	if err != nil {
		logger.Error(err)
	}
}

func hookCloudflare(env *env.Environment) error {
	if clearance != "" {
		return nil
	}

	baseUrl := env.GetString("browser-less.reversal")
	if !env.GetBool("browser-less.enabled") && baseUrl == "" {
		return errors.New("trying cloudflare failed, please setting `browser-less.enabled` or `browser-less.reversal`")
	}

	logger.Info("trying cloudflare ...")

	mu.Lock()
	defer mu.Unlock()
	if clearance != "" {
		return nil
	}

	if baseUrl == "" {
		baseUrl = "http://127.0.0.1:" + env.GetString("browser-less.port")
	}

	r, err := emit.ClientBuilder(common.HTTPClient).
		GET(baseUrl+"/clearance").
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		logger.Error(err)
		if emit.IsJSON(r) == nil {
			logger.Error(emit.TextResponse(r))
		}
		return err
	}

	defer r.Body.Close()
	obj, err := emit.ToMap(r)
	if err != nil {
		logger.Error(err)
		return err
	}

	data := obj["data"].(map[string]interface{})
	clearance = data["cookie"].(string)
	userAgent = data["userAgent"].(string)
	lang = data["lang"].(string)
	return nil
}

func cleanCloudflare() {
	mu.Lock()
	clearance = ""
	mu.Unlock()
}

func elseOf[T any](obj any) (zero T) {
	if obj == nil {
		return
	}
	return obj.(T)
}
