package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/logger"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/emit.io"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const GOOGLE_BASE_FORMAT = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s"

type funcDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      struct {
		Properties map[string]interface{} `json:"properties"`
		Required   []string               `json:"required"`
		Type       string                 `json:"type"`
	} `json:"parameters"`
}

// 构建请求，返回响应
func build(ctx context.Context, proxies, token string, messages []map[string]interface{}, completion pkg.ChatCompletion) (*http.Response, error) {
	gURL := fmt.Sprintf(GOOGLE_BASE_FORMAT, completion.Model, token)

	if completion.Temperature < 0.1 {
		completion.Temperature = 1
	}

	if completion.MaxTokens == 0 {
		completion.MaxTokens = 2048
	}

	if completion.TopK == 0 {
		completion.TopK = 100
	}

	if completion.TopP == 0 {
		completion.TopP = 0.95
	}

	// 参数基本与openai对齐
	_funcDecls := make([]funcDecl, 0)
	if toolsL := len(completion.Tools); toolsL > 0 {
		for _, v := range completion.Tools {
			kv := v.GetKeyv("function").GetKeyv("parameters")
			required, ok := kv.Get("required")
			if !ok {
				required = []string{}
			}

			_funcDecls = append(_funcDecls, funcDecl{
				Name:        strings.Replace(v.GetKeyv("function").GetString("name"), "-", "_", -1),
				Description: v.GetKeyv("function").GetString("description"),
				Params: struct {
					Properties map[string]interface{} `json:"properties"`
					Required   []string               `json:"required"`
					Type       string                 `json:"type"`
				}{
					Properties: kv.GetKeyv("properties"),
					Required:   required.([]string),
					Type:       kv.GetString("function"),
				},
			})
		}
	}

	// fix: Please ensure that multiturn requests ends with a user role or a function response.
	if messages[0]["role"] != "user" {
		messages = append([]map[string]interface{}{
			{
				"role": "user",
				"parts": []interface{}{
					map[string]string{
						"text": "hi ~",
					},
				},
			},
		}, messages...)
	}

	payload := map[string]any{
		"contents": messages, // [ { role: user, parts: [ { text: 'xxx' } ] } ]
		"generationConfig": map[string]any{
			"topK":            completion.TopK,
			"topP":            completion.TopP,
			"temperature":     completion.Temperature, // 0.8
			"maxOutputTokens": completion.MaxTokens,
			"stopSequences":   []string{},
		},
		// 安全级别
		"safetySettings": []map[string]string{
			{
				"category":  "HARM_CATEGORY_HARASSMENT",
				"threshold": "BLOCK_NONE",
			},
			{
				"category":  "HARM_CATEGORY_HATE_SPEECH",
				"threshold": "BLOCK_NONE",
			},
			{
				"category":  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				"threshold": "BLOCK_NONE",
			},
			{
				"category":  "HARM_CATEGORY_DANGEROUS_CONTENT",
				"threshold": "BLOCK_NONE",
			},
		},
	}

	if len(_funcDecls) > 0 {
		// 函数调用
		payload["tools"] = []map[string]interface{}{
			{
				"function_declarations": _funcDecls,
			},
		}
	}
	marshal, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	res, err := emit.ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		POST(gURL).
		JHeader().
		Bytes(marshal).
		Do()
	if err != nil {
		logger.Error(err)
		var e *url.Error
		if errors.As(err, &e) {
			e.URL = strings.Replace(e.URL, token, "AIzaSy***", -1)
		}
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		h := res.Header
		if c := h.Get("content-type"); strings.Contains(c, "application/json") {
			bts, e := io.ReadAll(res.Body)
			if e == nil {
				return nil, fmt.Errorf("%s: %s", res.Status, bts)
			}
		}
		return nil, errors.New(res.Status)
	}

	return res, nil
}
