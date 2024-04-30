package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	com "github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/gio.emits/common"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bincooo/goole15"
)

const MODEL = "gemini"
const GOOGLE_BASE = "https://generativelanguage.googleapis.com/%s?alt=sse&key=%s"
const login = "http://127.0.0.1:8081/v1/login"

var (
	// TODO clear loop
	gkv = make(map[uint32]cookieOpts)
	mu  sync.Mutex

	okey = "okey!"
)

type cookieOpts struct {
	userAgent string
	cookie    string
}

type candidatesResponse struct {
	Candidates []candidate `json:"candidates"`
}

type candidate struct {
	Content struct {
		Role  string                   `json:"role"`
		Parts []map[string]interface{} `json:"parts"`
	} `json:"content"`
	FinishReason string `json:"finishReason"`
	Index        int    `json:"index"`
}

// https://ai.google.dev/models/gemini?hl=zh-cn
func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []com.Matcher) {
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)

	messageL := len(req.Messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, -1, "[] is too short - 'messages'")
		return
	}

	messages, tokens, err := buildConversation(req.Messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	ctx.Set("tokens", tokens)

	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	response, err := build(ctx.Request.Context(), proxies, cookie, messages, req)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}
	waitResponse(ctx, matchers, response, req.Stream)
}

// https://ai.google.dev/models/gemini?hl=zh-cn
func Complete15(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []com.Matcher) {
	var (
		token   = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)

	messageL := len(req.Messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, -1, "[] is too short - 'messages'")
		return
	}

	messages, tokens, err := buildConversation15(req.Messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	ctx.Set("tokens", tokens)

	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	// 解析cookie
	sign, auth, key, user, co, err := extCookie15(ctx.Request.Context(), token, proxies)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	opts := goole.NewDefaultOptions(proxies)
	opts.Temperature(req.Temperature)
	opts.TopP(req.TopP)
	opts.TopK(req.TopK)
	h := com.Hash(token)
	if c, ok := gkv[h]; ok {
		opts.UA(c.userAgent)
	}

	chat := goole.New(co, sign, auth, key, user, opts)
	ch, err := chat.Reply(ctx.Request.Context(), messages)
	if err != nil {
		code := -1
		errMessage := err.Error()
		if strings.Contains(errMessage, "429 Too Many Requests") {
			code = http.StatusTooManyRequests
		}
		if strings.Contains(errMessage, "500 Internal Server Error") {
			delete(gkv, h) // 尚不清楚 500 错误的原因
		}
		middle.ResponseWithE(ctx, code, err)
		return
	}
	waitResponse15(ctx, matchers, ch, req.Stream)
}

func extCookie15(ctx context.Context, token, proxies string) (sign, auth, key, user string, cookie string, err error) {
	var opts cookieOpts
	h := com.Hash(token)

	if !strings.Contains(token, "@gmail.com|") {
		// 不走接口获取的token
		opts = cookieOpts{
			cookie: token,
		}
		//
	} else if co, ok := gkv[h]; ok {
		opts = co
		logrus.Info("cookie: ", co.cookie)
	} else {
		s := strings.Split(token, "|")
		if len(s) < 4 {
			err = errors.New("invalid token")
			return
		}

		gLogin := pkg.Config.GetString("goole")
		if gLogin == "" {
			gLogin = login
		}

		mu.Lock()
		defer mu.Unlock()

		timeout, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		response, e := common.ClientBuilder().
			Proxies(proxies).
			POST(gLogin).
			Context(timeout).
			Header("Authorization", s[3]).
			Body(map[string]string{
				"mail":   s[0],
				"cMail":  s[1],
				"passwd": s[2],
			}).
			JHeader().
			DoWith(http.StatusOK)
		if e != nil {
			err = fmt.Errorf("fetch cookies failed: %v", e)
			return
		}

		var result map[string]interface{}
		e = common.ToObject(response, &result)
		if e != nil {
			err = errors.New(fmt.Sprintf("fetch cookies failed: %v", e))
			return
		}

		if !reflect.DeepEqual(result["ok"], true) {
			err = errors.New(fmt.Sprintf("fetch cookies failed: %s", result["message"]))
			return
		}

		opts = cookieOpts{
			userAgent: result["userAgent"].(string),
			cookie:    result["cookies"].(string),
		}
		gkv[h] = opts
	}

	cookie = opts.cookie
	logrus.Info("cookie: ", cookie)
	index := strings.Index(cookie, "[sign=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			sign = cookie[index+6 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}

	index = strings.Index(cookie, "[auth=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			auth = cookie[index+6 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}

	index = strings.Index(cookie, "[key=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			key = cookie[index+5 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}

	index = strings.Index(cookie, "[u=")
	if index > -1 {
		end := strings.Index(cookie[index:], "]")
		if end > -1 {
			user = cookie[index+3 : index+end]
			cookie = cookie[:index] + cookie[index+end+1:]
		}
	}
	return
}

func waitResponse(ctx *gin.Context, matchers []com.Matcher, partialResponse *http.Response, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")
	tokens := ctx.GetInt("tokens")

	reader := bufio.NewReader(partialResponse.Body)
	var original []byte
	var block = []byte("data: ")
	var functionCall interface{}
	isError := false

	for {
		line, hm, err := reader.ReadLine()
		original = append(original, line...)
		if hm {
			continue
		}

		if err == io.EOF {
			if isError {
				middle.ResponseWithV(ctx, -1, string(original))
				return
			}
			break
		}

		if err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}

		if len(original) == 0 {
			continue
		}

		if isError {
			continue
		}

		if bytes.Contains(original, []byte(`"error":`)) {
			isError = true
			continue
		}

		if !bytes.HasPrefix(original, block) {
			continue
		}

		var c candidatesResponse
		original = bytes.TrimPrefix(original, block)
		if err = json.Unmarshal(original, &c); err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}

		cond := c.Candidates[0]
		if cond.Content.Role != "model" {
			original = nil
			continue
		}

		if fc, ok := cond.Content.Parts[0]["functionCall"]; ok {
			functionCall = fc
			original = nil
			continue
		}

		raw, ok := cond.Content.Parts[0]["text"]
		if !ok {
			original = nil
			continue
		}
		fmt.Printf("----- raw -----\n %s\n", raw)
		original = nil
		raw = com.ExecMatchers(matchers, raw.(string))

		if sse {
			middle.ResponseWithSSE(ctx, MODEL, raw.(string), nil, created)
		}
		content += raw.(string)

	}

	if functionCall != nil {
		fc := functionCall.(map[string]interface{})
		args, _ := json.Marshal(fc["args"])
		if sse {
			middle.ResponseWithSSEToolCalls(ctx, MODEL, fc["name"].(string), string(args), created)
		} else {
			middle.ResponseWithToolCalls(ctx, MODEL, fc["name"].(string), string(args))
		}
		return
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", com.CalcUsageTokens(content, tokens), created)
	}
}

func waitResponse15(ctx *gin.Context, matchers []com.Matcher, ch chan string, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")
	tokens := ctx.GetInt("tokens")

	for {
		tex, ok := <-ch
		if !ok {
			break
		}

		if strings.HasPrefix(tex, "error: ") {
			message := strings.TrimPrefix(tex, "error: ")
			code := -1
			if strings.Contains(message, "429 Too Many Requests") {
				code = http.StatusTooManyRequests
			}
			middle.ResponseWithV(ctx, code, message)
			return
		}

		if strings.HasPrefix(tex, "text: ") {
			raw := strings.TrimPrefix(tex, "text: ")
			fmt.Printf("----- raw -----\n %s\n", raw)
			raw = com.ExecMatchers(matchers, raw)
			if sse {
				middle.ResponseWithSSE(ctx, MODEL+"-1.5", raw, nil, created)
			}
			content += raw

		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL+"-1.5", content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL+"-1.5", "[DONE]", com.CalcUsageTokens(content, tokens), created)
	}
}

func buildConversation(messages []map[string]string) (newMessages []map[string]interface{}, tokens int, err error) {
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	pos = 0
	messageL := len(messages)

	role := ""
	buffer := make([]string, 0)

	mergeMessages := make([]map[string]string, 0)
	// role类型转换
	condition := func(expr string) string {
		switch expr {
		case "user", "function":
			return "user"
		case "system":
			return expr
		case "assistant":
			return "model"
		default:
			return ""
		}
	}

	push := func(pos int, role string, content string) {
		if role == "system" {
			mergeMessages = append(mergeMessages, map[string]string{
				"role":    "user",
				"content": content,
			})
			mergeMessages = append(mergeMessages, map[string]string{
				"role":    "model",
				"content": okey,
			})
		} else {
			mergeMessages = append(mergeMessages, map[string]string{
				"role":    role,
				"content": content,
			})
		}
	}

	// 合并历史对话
	// merge one
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				join := strings.Join(buffer, "\n\n")
				if len(strings.TrimSpace(join)) > 0 {
					push(pos, role, join)
				}
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return nil, -1, errors.New(
				fmt.Sprintf("'%s' is not one of ['system', 'assistant', 'user', 'function'] - 'messages.%d.role'",
					message["role"], pos))
		}
		pos++
		if role == "" {
			role = curr
		}

		if message["role"] == "function" {
			content = fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], content)
		}

		if curr == role {
			if len(strings.TrimSpace(content)) > 0 {
				buffer = append(buffer, content)
			}
			continue
		}

		join := strings.Join(buffer, "\n\n")
		if len(join) > 0 {
			push(pos, role, join)
		}

		buffer = append(make([]string, 0), content)
		role = curr
	}

	push = func(pos int, role string, content string) {
		if role == "system" {
			newMessages = append(newMessages, map[string]interface{}{
				"role": "user",
				"parts": []interface{}{
					map[string]string{
						"text": content,
					},
				},
			})
		} else {
			newMessages = append(newMessages, map[string]interface{}{
				"role": role,
				"parts": []interface{}{
					map[string]string{
						"text": content,
					},
				},
			})
		}
	}

	pos = 0
	role = ""
	buffer = make([]string, 0)
	messageL = len(mergeMessages)

	// [ { role: user, parts: [ { text: 'xxx' } ] } ]
	// merge two
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				join := strings.Join(buffer, "\n\n")
				if len(join) > 0 {
					tokens += com.CalcTokens(join)
					push(pos, role, join)
				}
			}

			message := newMessages[len(newMessages)-1]
			if message["role"] == "model" { //
				newMessages = append(newMessages, map[string]interface{}{
					"role": "user",
					"parts": []interface{}{
						map[string]string{
							"text": "continue",
						},
					},
				})
			}
			break
		}

		message := mergeMessages[pos]
		curr := message["role"]
		content := message["content"]

		pos++
		if role == "" {
			role = curr
		}

		if curr == role {
			buffer = append(buffer, content)
			continue
		}

		join := strings.Join(buffer, "\n\n")
		if len(join) > 0 {
			tokens += com.CalcTokens(join)
			push(pos, role, join)
		}

		buffer = append(make([]string, 0), content)
		role = curr
	}
	return
}

func buildConversation15(messages []map[string]string) ([]goole.Message, int, error) {
	pos := len(messages) - 1
	if pos < 0 {
		return nil, -1, errors.New("messages is empty")
	}

	pos = 0
	messageL := len(messages)
	tokens := 0

	role := ""
	buffer := make([]string, 0)

	var newMessages []goole.Message

	condition := func(expr string) string {
		switch expr {
		case "user", "system", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	// 检查下一个消息体是否是指定的role类型
	next := func(pos int, role string) bool {
		if pos+1 >= messageL {
			return false
		}
		message := messages[pos+1]
		return condition(message["role"]) == role
	}

	push := func(pos int, role string, content string) {
		if role == "system" {
			newMessages = append(newMessages, goole.Message{
				Role:    role,
				Content: content,
			})

			if !next(pos, "assistant") {
				newMessages = append(newMessages, goole.Message{
					Role:    "assistant",
					Content: okey,
				})
			}
		} else {
			newMessages = append(newMessages, goole.Message{
				Role:    role,
				Content: content,
			})
		}
	}

	// 合并历史对话
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				join := strings.Join(buffer, "\n\n")
				if len(join) > 0 {
					tokens += com.CalcTokens(join)
					push(pos, role, join)
				}
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return nil, -1, errors.New(
				fmt.Sprintf("'%s' is not one of ['system', 'assistant', 'user', 'function'] - 'messages.%d.role'",
					message["role"], pos))
		}
		pos++
		if role == "" {
			role = curr
		}

		if curr == "function" {
			content = fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], content)
		}

		if curr == role {
			buffer = append(buffer, content)
			continue
		}

		join := strings.Join(buffer, "\n\n")
		if len(join) > 0 {
			tokens += com.CalcTokens(join)
			push(pos, role, join)
		}

		buffer = append(make([]string, 0), content)
		role = curr
	}

	if newMessages[0].Role != "user" {
		newMessages = append([]goole.Message{
			{
				Role:    "user",
				Content: "hi ~",
			},
		}, newMessages...)
	}
	return newMessages, tokens, nil
}

//
//
//
