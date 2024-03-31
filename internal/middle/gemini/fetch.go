package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/sirupsen/logrus"
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
func build(ctx context.Context, proxies, token, content string, req gpt.ChatCompletionRequest) (*http.Response, error) {
	var (
		burl = fmt.Sprintf(GOOGLE_BASE, "v1beta/models/gemini-1.0-pro:streamGenerateContent", token)
	)

	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}

	if req.TopK == 0 {
		req.TopK = 1
	}

	if req.TopP == 0 {
		req.TopP = 1
	}

	// 参数基本与openai对齐
	_funcDecl := make([]funcDecl, 0)
	if toolsL := len(req.Tools); toolsL > 0 {
		for _, v := range req.Tools {
			_funcDecl = append(_funcDecl, funcDecl{
				Name:        strings.Replace(v.Fun.Name, "-", "_", -1),
				Description: v.Fun.Description,
				Params:      v.Fun.Params,
			})
		}
	}

	marshal, err := json.Marshal(map[string]any{
		"contents": struct {
			Parts []map[string]string `json:"parts"`
		}{[]map[string]string{
			{"text": content},
		}}, // [ { role: user, parts: [ 'xxx' ] } ]
		"generationConfig": map[string]any{
			"topK":            req.TopK,
			"topP":            req.TopP,
			"temperature":     req.Temperature, // 0.8
			"maxOutputTokens": req.MaxTokens,
			"stopSequences":   []string{},
		},
		// 函数调用
		"tools": []map[string][]any{
			{
				"function_declarations": []any{
					_funcDecl,
				},
			},
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
	})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, burl, bytes.NewReader(marshal))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	client, err := common.NewHttpClient(proxies)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	res, err := client.Do(request.WithContext(ctx))
	if err != nil {
		logrus.Error(err)
		var e *url.Error
		if errors.As(err, &e) {
			e.URL = strings.Replace(e.URL, token, "AIzaSy***", -1)
		}
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}

	return res, nil
}
