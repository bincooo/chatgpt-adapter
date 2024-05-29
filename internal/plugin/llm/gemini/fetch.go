package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const GOOGLE_BASE_FORMAT = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s"

var safetySettings = []map[string]interface{}{
	//{
	//	"category":  "HARM_CATEGORY_DEROGATORY",
	//	"threshold": "BLOCK_NONE",
	//},
	//{
	//	"category":  "HARM_CATEGORY_TOXICITY",
	//	"threshold": "BLOCK_NONE",
	//},
	//{
	//	"category":  "HARM_CATEGORY_VIOLENCE",
	//	"threshold": "BLOCK_NONE",
	//},
	//{
	//	"category":  "HARM_CATEGORY_SEXUAL",
	//	"threshold": "BLOCK_NONE",
	//},
	//{
	//	"category":  "HARM_CATEGORY_DANGEROUS",
	//	"threshold": "BLOCK_NONE",
	//},
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
}

type funcDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      struct {
		Properties map[string]interface{} `json:"properties"`
		Required   []string               `json:"required"`
		Type       string                 `json:"type"`
	} `json:"parameters"`
}

func init() {
	common.AddInitialized(initSafetySettings)
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

	toStrings := func(slice []interface{}) (values []string) {
		for _, v := range slice {
			values = append(values, v.(string))
		}
		return
	}

	condition := func(str string) string {
		switch str {
		case "string":
			return "STRING"
		case "boolean":
			return "BOOLEAN"
		case "number":
			return "NUMBER"
		default:
			if strings.HasPrefix(str, "array") {
				return "ARRAY"
			}
			return "OBJECT"
		}
	}

	// fix: type 枚举必须符合google定义，否则报错400
	// https://ai.google.dev/api/rest/v1beta/Schema?hl=zh-cn#type
	var fix func(keyv pkg.Keyv[interface{}]) pkg.Keyv[interface{}]
	{
		fix = func(keyv pkg.Keyv[interface{}]) pkg.Keyv[interface{}] {
			if keyv.Has("type") {
				keyv.Set("type", condition(keyv.GetString("type")))
			}
			for k, _ := range keyv {
				child := keyv.GetKeyv(k)
				if child != nil {
					keyv.Set(k, fix(child))
				}
			}
			return keyv
		}
	}

	// 参数基本与openai对齐
	_funcDecls := make([]funcDecl, 0)
	if toolsL := len(completion.Tools); toolsL > 0 {
		for _, v := range completion.Tools {
			kv := v.GetKeyv("function").GetKeyv("parameters")
			required := kv.GetSlice("required")
			_funcDecls = append(_funcDecls, funcDecl{
				// 必须为 a-z、A-Z、0-9，或包含下划线和短划线，长度上限为 63 个字符
				Name:        strings.Replace(v.GetKeyv("function").GetString("name"), "-", "_", -1),
				Description: v.GetKeyv("function").GetString("description"),
				Params: struct {
					Properties map[string]interface{} `json:"properties"`
					Required   []string               `json:"required"`
					Type       string                 `json:"type"`
				}{
					Properties: fix(kv.GetKeyv("properties")),
					Required:   toStrings(required),
					Type:       condition(kv.GetString("type")),
				},
			})
		}
	}

	// 获取top system作为systemInstruction
	system := ""
	if messages[0]["role"] == "system" {
		if content, ok := messages[0]["content"].(string); ok {
			messages = messages[1:]
			system = content
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
		"safetySettings": safetySettings,
	}

	if len(system) != 0 {
		payload["systemInstruction"] = map[string]interface{}{
			"role": "system",
			"parts": []interface{}{
				map[string]string{
					"text": system,
				},
			},
		}
	}

	if len(_funcDecls) > 0 {
		// 函数调用
		payload["tools"] = []map[string]interface{}{
			{
				"functionDeclarations": _funcDecls,
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
		if c := h.Get("content-type"); !strings.Contains(c, "text/html") {
			dataBytes, e := io.ReadAll(res.Body)
			if e == nil {
				logger.Errorf("%s", dataBytes)
			}
		}
		return nil, errors.New(res.Status)
	}

	return res, nil
}

func initSafetySettings() {
	harmBlocks := []string{
		"HARM_BLOCK_THRESHOLD_UNSPECIFIED",
		"BLOCK_LOW_AND_ABOVE",
		"BLOCK_MEDIUM_AND_ABOVE",
		"BLOCK_ONLY_HIGH",
		"BLOCK_NONE",
	}

	if safes := pkg.Config.Get("google.safes"); safes != nil {
		values, ok := safes.([]interface{})
		pass := true
		if !ok {
			return
		}

		tempSafetySettings := make([]map[string]interface{}, 0)
		for _, value := range values {
			var (
				category  = ""
				threshold = ""
			)

			safe, okey := value.(map[string]interface{})
			if !okey {
				logger.Errorf("Failed to parse safety settings: %v", value)
				return
			}

			if v, o := safe["category"]; o {
				category = fmt.Sprintf("%s", v)
			}
			if v, o := safe["threshold"]; o {
				threshold = fmt.Sprintf("%s", v)
			}

			for _, setting := range safetySettings {
				if setting["category"] == category {
					if !slices.Contains(harmBlocks, threshold) {
						logger.Errorf("%s is not in %+v", threshold, harmBlocks)
						pass = false
						break
					}
					tempSafetySettings = append(tempSafetySettings, safe)
				}
			}
			if !pass {
				return
			}
		}
		safetySettings = tempSafetySettings
	}
}
