package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	com "github.com/bincooo/goole15/common"
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
const GOOGLE_BASE = "https://generativelanguage.googleapis.com/%s?key=%s"
const login = "http://127.0.0.1:8081/v1/login"

// TODO clear loop
var gkv = make(map[uint32]cookieOpts)
var mu sync.Mutex

type cookieOpts struct {
	userAgent string
	cookie    string
}

func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []common.Matcher) {
	var (
		cookie  = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)

	// 复原转码
	matchers = appendMatchers(matchers)

	messages := req.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, -1, "[] is too short - 'messages'")
		return
	}

	content, err := buildConversation(messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	response, err := build(ctx.Request.Context(), proxies, cookie, content, req)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}
	waitResponse(ctx, matchers, response, req.Stream)
}

func Complete15(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []common.Matcher) {
	var (
		token   = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)

	// 复原转码
	matchers = appendMatchers(matchers)
	messages := req.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, -1, "[] is too short - 'messages'")
		return
	}

	content, err := buildConversation15(messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	// 解析cookie
	sign, auth, key, user, co, err := extCookie(ctx.Request.Context(), token, proxies)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	opts := goole.NewDefaultOptions(proxies)
	opts.UA(gkv[common.Hash(token)].userAgent)
	chat := goole.New(co, sign, auth, key, user, opts)
	ch, err := chat.Reply(ctx.Request.Context(), content)
	if err != nil {
		code := -1
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			code = http.StatusTooManyRequests
		}
		middle.ResponseWithE(ctx, code, err)
		return
	}
	waitResponse15(ctx, matchers, ch, req.Stream)
}

func extCookie(ctx context.Context, token, proxies string) (sign, auth, key, user string, cookie string, err error) {
	var opts cookieOpts
	h := common.Hash(token)
	if co, ok := gkv[h]; ok {
		opts = co
	} else {
		s := strings.Split(token, "|")
		if len(s) < 4 {
			err = errors.New("invalid token")
			return
		}

		mu.Lock()
		defer mu.Unlock()
		response, e := com.New().
			Proxies(proxies).
			URL(login).
			Method(http.MethodPost).
			Context(ctx).
			Header("Authorization", s[3]).
			SetBody(map[string]string{
				"mail":   s[0],
				"cMail":  s[1],
				"passwd": s[2],
			}).
			JsonHeader().
			Do()
		if e != nil {
			err = errors.New("fetch cookies failed")
			return
		}

		if response.StatusCode != http.StatusOK {
			err = errors.New("fetch cookies failed: " + response.Status)
			return
		}

		var result map[string]interface{}
		e = com.ToObj(response, &result)
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

func appendMatchers(matchers []common.Matcher) []common.Matcher {
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "*",
		H: func(index int, content string) (state int, result string) {
			// 换行符处理
			content = strings.ReplaceAll(content, `\n`, "\n")
			// <符处理
			idx := strings.Index(content, "\\u003c")
			for idx >= 0 {
				content = content[:idx] + "<" + content[idx+6:]
				idx = strings.Index(content, "\\u003c")
			}
			// >符处理
			idx = strings.Index(content, "\\u003e")
			for idx >= 0 {
				content = content[:idx] + ">" + content[idx+6:]
				idx = strings.Index(content, "\\u003e")
			}
			// "符处理
			content = strings.ReplaceAll(content, `\"`, "\"")
			return common.MAT_MATCHED, content
		},
	})
	return matchers
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, partialResponse *http.Response, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")

	reader := bufio.NewReader(partialResponse.Body)
	var original []byte
	var block = []byte(`"text": "`)
	var fBlock = []byte(`"functionCall": {`)
	isError := false
	isFunc := false

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

		if isFunc {
			continue
		}

		if bytes.Contains(original, []byte(`"error":`)) {
			isError = true
			continue
		}

		if bytes.Contains(original, fBlock) {
			isFunc = true
			continue
		}

		if !bytes.Contains(original, block) {
			continue
		}

		index := bytes.Index(original, block)
		raw := string(original[index+len(block) : len(original)-1])
		fmt.Printf("----- raw -----\n %s\n", raw)
		original = make([]byte, 0)
		raw = common.ExecMatchers(matchers, raw)

		if sse {
			middle.ResponseWithSSE(ctx, MODEL, raw, created)
		} else {
			content += raw
		}

	}

	if isFunc {
		var dict []map[string]any
		err := json.Unmarshal(original, &dict)
		if err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}

		candidate := dict[0]["candidates"].([]interface{})[0].(map[string]interface{})
		cont := candidate["content"].(map[string]interface{})
		part := cont["parts"].([]interface{})[0].(map[string]interface{})
		functionCall := part["functionCall"].(map[string]interface{})

		indent, err := json.MarshalIndent(functionCall["args"], "", "")
		if err != nil {
			middle.ResponseWithE(ctx, -1, err)
			return
		}

		name := functionCall["name"].(string)
		index := strings.Index(name, "_")
		name = name[:index] + "-" + name[index+1:]

		if sse {
			middle.ResponseWithSSEToolCalls(ctx, MODEL, name, string(indent), created)
		} else {
			middle.ResponseWithToolCalls(ctx, MODEL, name, string(indent))
		}
		return
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", created)
	}
}

func waitResponse15(ctx *gin.Context, matchers []common.Matcher, ch chan string, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")

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
			raw = common.ExecMatchers(matchers, raw)
			if sse {
				middle.ResponseWithSSE(ctx, MODEL, raw, created)
			} else {
				content += raw
			}
		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", created)
	}
}

func buildConversation(messages []map[string]string) (string, error) {
	pos := len(messages) - 1
	if pos < 0 {
		return "", nil
	}

	pos = 0
	messageL := len(messages)

	role := ""
	buffer := make([]string, 0)

	condition := func(expr string) string {
		switch expr {
		case "system", "function", "assistant":
			return expr
		case "user":
			return "human"
		default:
			return ""
		}
	}

	pMessages := ""

	// 合并历史对话
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessages += fmt.Sprintf("%s:\n %s\n\n", strings.Title(role), strings.Join(buffer, "\n\n"))
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return "", errors.New(
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
		pMessages += fmt.Sprintf("%s: \n%s\n\n", strings.Title(role), strings.Join(buffer, "\n\n"))
		buffer = append(make([]string, 0), content)
		role = curr
	}

	return pMessages, nil
}

func buildConversation15(messages []map[string]string) ([]goole.Message, error) {
	pos := len(messages) - 1
	if pos < 0 {
		return nil, errors.New("messages is empty")
	}

	pos = 0
	messageL := len(messages)

	role := ""
	buffer := make([]string, 0)

	condition := func(expr string) string {
		switch expr {
		case "user", "system", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	var pMessages []goole.Message

	// 合并历史对话
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessages = append(pMessages, goole.Message{
					Role:    role,
					Content: strings.Join(buffer, "\n\n"),
				})
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return nil, errors.New(
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

		pMessages = append(pMessages, goole.Message{
			Role:    role,
			Content: strings.Join(buffer, "\n\n"),
		})
		buffer = append(make([]string, 0), content)
		role = curr
	}

	return pMessages, nil
}

//
//
//
