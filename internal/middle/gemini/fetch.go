package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/gio.emits/common"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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
func build(ctx context.Context, proxies, token string, messages []map[string]interface{}, req gpt.ChatCompletionRequest) (*http.Response, error) {
	var (
		burl = fmt.Sprintf(GOOGLE_BASE, "v1beta/models/gemini-1.0-pro-latest:streamGenerateContent", token)
	)

	if req.Model == "gemini-1.5" {
		burl = fmt.Sprintf(GOOGLE_BASE, "v1beta/models/gemini-1.5-pro-latest:streamGenerateContent", token)
	}

	if req.Temperature < 0.1 {
		req.Temperature = 1
	}

	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}

	if req.TopK == 0 {
		req.TopK = 100
	}

	if req.TopP == 0 {
		req.TopP = 0.95
	}

	// 参数基本与openai对齐
	_funcDecls := make([]funcDecl, 0)
	if toolsL := len(req.Tools); toolsL > 0 {
		for _, v := range req.Tools {
			_funcDecls = append(_funcDecls, funcDecl{
				Name:        strings.Replace(v.Fun.Name, "-", "_", -1),
				Description: v.Fun.Description,
				Params:      v.Fun.Params,
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
			"topK":            req.TopK,
			"topP":            req.TopP,
			"temperature":     req.Temperature, // 0.8
			"maxOutputTokens": req.MaxTokens,
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
		logrus.Error(err)
		return nil, err
	}

	res, err := common.ClientBuilder().
		Proxies(proxies).
		Context(ctx).
		POST(burl).
		JHeader().
		Bytes(marshal).
		Do()
	if err != nil {
		logrus.Error(err)
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
