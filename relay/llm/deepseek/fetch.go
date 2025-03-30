package deepseek

import (
	"bytes"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1.1 Safari/605.1.15"
	lang      string
	clearance string

	mu sync.Mutex

	calcServer = "https://wik5ez2o-helper.hf.space"
)

//var (
//	wasmInstance wasm.Instance
//)
//
//func init() {
//	inst, err := wasm.New("./relay/llm/deepseek/sha3_wasm_bg.wasm")
//	if err != nil {
//		panic(err)
//	}
//	wasmInstance = inst
//}

type deepseekRequest struct {
	ChatSessionId   string `json:"chat_session_id"`
	ParentMessageId *int   `json:"parent_message_id"`
	Message         string `json:"prompt"`
	RefFileIds      []int  `json:"ref_file_ids"`
	ThinkingEnabled bool   `json:"thinking_enabled"`
	SearchEnabled   bool   `json:"search_enabled"`
}

func fetch(ctx context.Context, proxied, cookie string, request deepseekRequest) (response *http.Response, err error) {
	retry := 3
label:
	retry--
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST("https://chat.deepseek.com/api/v0/chat/create_pow_challenge").
		JSONHeader().
		Ja3().
		Header("authorization", "Bearer "+cookie).
		Header("referer", "https://chat.deepseek.com/a/chat/").
		Header("user-agent", userAgent).
		Header("x-app-version", "20241129.1").
		Header("x-client-locale", "zh_CN").
		Header("x-client-platform", "web").
		Header("x-client-version", "1.0.0-always").
		Header(elseOf(clearance != "", "cookie"), clearance).
		Header(elseOf(lang != "", "accept-language"), lang).
		Body(map[string]interface{}{
			"target_path": "/api/v0/chat/completion",
		}).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}

	obj, err := emit.ToMap(response)
	if err != nil {
		_ = response.Body.Close()
		return
	}

	_ = response.Body.Close()
	if code, ok := obj["code"]; !ok || code.(float64) != 0 {
		err = fmt.Errorf("create challenge failed")
		msg := obj["msg"]
		if msg != "" {
			err = fmt.Errorf(msg.(string))
		}
		return
	}

	value, ok := obj["data"].(map[string]interface{})["biz_data"]
	if !ok {
		err = fmt.Errorf("create challenge failed")
		return
	}

	data := value.(map[string]interface{})
	data = data["challenge"].(map[string]interface{})
	num, err := calcAnswer(data)
	if err != nil {
		return
	}

	buf, err := json.Marshal(map[string]interface{}{
		"algorithm":   "DeepSeekHashV1",
		"challenge":   data["challenge"],
		"salt":        data["salt"],
		"answer":      num,
		"signature":   data["signature"],
		"target_path": "/api/v0/chat/completion",
	})
	if err != nil {
		return
	}

	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST("https://chat.deepseek.com/api/v0/chat/completion").
		JSONHeader().
		Ja3().
		Header("authorization", "Bearer "+cookie).
		Header("origin", "https://chat.deepseek.com").
		Header("referer", "https://chat.deepseek.com/").
		Header("user-agent", userAgent).
		Header("x-app-version", "20241129.1").
		Header("x-client-locale", "zh_CN").
		Header("x-client-platform", "web").
		Header("x-client-version", "1.0.0-always").
		Header("x-ds-pow-response", base64.RawStdEncoding.EncodeToString(buf)).
		Header(elseOf(clearance != "", "cookie"), clearance).
		Header(elseOf(lang != "", "accept-language"), lang).
		Body(request).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		var busErr emit.Error
		if errors.As(err, &busErr) && strings.Contains(busErr.Msg, "code\":40300,\"msg\":\"Missing Header") {
			if retry > 0 {
				logger.Error(err)
				goto label
			}
		}
	}
	return
}

func deleteSession(ctx *gin.Context, env *env.Environment, sessionId string) {
	_, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(env.GetString("server.proxied")).
		POST("https://chat.deepseek.com/api/v0/chat_session/delete").
		JSONHeader().
		Ja3().
		Header("authorization", "Bearer "+ctx.GetString("token")).
		Header("referer", "https://chat.deepseek.com/").
		Header("user-agent", userAgent).
		Header("x-app-version", "20241129.1").
		Header("x-client-locale", "zh_CN").
		Header("x-client-platform", "web").
		Header("x-client-version", "1.0.0-always").
		Header(elseOf(clearance != "", "cookie"), clearance).
		Header(elseOf(lang != "", "accept-language"), lang).
		Body(map[string]interface{}{
			"chat_session_id": sessionId,
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		logger.Error(err)
		return
	}
}

func calcAnswer(data map[string]interface{}) (num int, err error) {
	timeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	challenge := data["challenge"].(string)
	salt := data["salt"].(string)
	diff := int(data["difficulty"].(float64))
	expireAt := int(data["expire_at"].(float64))
	r, err := emit.ClientBuilder(common.NopHTTPClient).
		Context(timeout).
		POST(calcServer+"/ds").
		JSONHeader().
		Body(map[string]interface{}{
			"challenge": challenge,
			"salt":      salt,
			"diff":      diff,
			"expireAt":  expireAt,
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}

	obj, err := emit.ToMap(r)
	if err != nil {
		return
	}

	if v, ok := obj["ok"]; !ok || v != true {
		err = fmt.Errorf("calc answer failed")
		return
	}
	num = int(obj["data"].(float64))
	return
}

//func calcAnswer(data map[string]interface{}) (num int, err error) {
//	__wbindgen_add_to_stack_pointer, err := wasmInstance.Exports.GetFunction("__wbindgen_add_to_stack_pointer")
//	if err != nil {
//		return
//	}
//	c, err := __wbindgen_add_to_stack_pointer(-16)
//	if err != nil {
//		return
//	}
//
//	memory, err := wasmInstance.Exports.GetMemory("memory")
//	if err != nil {
//		return
//	}
//	buffer := memory.Data()
//
//	u := func(e string, t, n wasm.NativeFunction) (i int, num int32, err error) {
//		//
//		r := len(e)
//		value, err := t(r, 1)
//		if err != nil {
//			return
//		}
//
//		num = value.(int32)
//		f := 0
//		for range r {
//			ch := e[f]
//			if ch > 127 {
//				break
//			}
//			buffer[int(ch)+f] = ch
//			f++
//		}
//		i = r
//		return
//	}
//
//	__wbindgen_export_1, err := wasmInstance.Exports.GetFunction("__wbindgen_export_1")
//	if err != nil {
//		return
//	}
//
//	// h := data["challenge"].(string)
//	h := "3530c39e5ee8a2c728fb0542fc80979e18dda94861499981d32b6d68f0d9eac7"
//	f, s, err := u(h, __wbindgen_export_0, __wbindgen_export_1)
//	if err != nil {
//		return
//	}
//
//	//t := fmt.Sprintf("%s_%d_", data["salt"], int(data["expire_at"].(float64)))
//	t := "cecc9bf94c68b3cfa920_1737771632818_"
//	p, d, err := u(t, __wbindgen_export_0, __wbindgen_export_1)
//	if err != nil {
//		return
//	}
//
//	wasm_solve, err := wasmInstance.Exports.GetFunction("wasm_solve")
//	if err != nil {
//		return
//	}
//
//	i := c.(int32)
//
//	//n1 := data["difficulty"].(float64)
//	var n1 float64 = 144000
//	_, err = wasm_solve(i, s, f, d, p, n1)
//	if err != nil {
//		return
//	}
//
//	defer __wbindgen_add_to_stack_pointer(16)
//	buf := buffer[i : i+4]
//	reader := bytes.NewReader(buf)
//	var n int32
//	err = binary.Read(reader, binary.LittleEndian, &n)
//	if err != nil {
//		return
//	}
//	buf = buffer[i+8 : i+16]
//	r := binary.BigEndian.Uint64(buf)
//	if n == 0 {
//		num = 0
//		return
//	}
//	num = int(r)
//	return
//}

func convertRequest(ctx *gin.Context, env *env.Environment, completion model.Completion) (request deepseekRequest, err error) {
	r, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(env.GetString("server.proxied")).
		POST("https://chat.deepseek.com/api/v0/chat_session/create").
		JSONHeader().
		Ja3().
		Header("authorization", "Bearer "+ctx.GetString("token")).
		Header("referer", "https://chat.deepseek.com/").
		Header("user-agent", userAgent).
		Header("x-app-version", "20241129.1").
		Header("x-client-locale", "zh_CN").
		Header("x-client-platform", "web").
		Header("x-client-version", "1.0.0-always").
		Header(elseOf(clearance != "", "cookie"), clearance).
		Header(elseOf(lang != "", "accept-language"), lang).
		Body(map[string]interface{}{
			"character_id": nil,
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		var busErr emit.Error
		if errors.As(err, &busErr) && busErr.Code == 403 {
			_ = hookCloudflare(env)
		}
		return
	}

	defer r.Body.Close()
	obj, err := emit.ToMap(r)
	if err != nil {
		return
	}

	if code, ok := obj["code"]; !ok || code.(float64) != 0 {
		err = fmt.Errorf("create chat session failed")
		msg := obj["msg"]
		if msg != "" {
			err = fmt.Errorf(msg.(string))
		}
		return
	}

	value, ok := obj["data"].(map[string]interface{})["biz_data"]
	if !ok {
		err = fmt.Errorf("create chat session failed")
		return
	}

	contentBuffer := new(bytes.Buffer)
	if len(completion.Messages) == 1 {
		contentBuffer.WriteString(completion.Messages[0].GetString("content"))
		goto label
	}

	for _, message := range completion.Messages {
		role, end := response.ConvertRole(ctx, message.GetString("role"))
		contentBuffer.WriteString(role)
		contentBuffer.WriteString(message.GetString("content"))
		contentBuffer.WriteString(end)
	}

label:
	data := value.(map[string]interface{})
	request = deepseekRequest{
		ChatSessionId:   data["id"].(string),
		RefFileIds:      make([]int, 0),
		ThinkingEnabled: completion.Model[9:] == "reasoner",
		SearchEnabled:   false,

		Message: contentBuffer.String(),
	}
	return
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
		GET(baseUrl+"/v0/clearance").
		Header("x-website", "https://chat.deepseek.com").
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

func elseOf[T any](condition bool, t T) (zero T) {
	if condition {
		return t
	}
	return zero
}
