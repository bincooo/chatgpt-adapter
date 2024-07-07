package gemini

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const GOOGLE_BASE_FORMAT = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s"

var (
	safetySettings = []map[string]interface{}{
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

	emp = map[string]interface{}{
		"-": map[string]string{
			"type": "string",
		},
	}
)

type funcDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      *struct {
		Properties map[string]interface{} `json:"properties,omitempty"`
		Required   []string               `json:"required,omitempty"`
		Type       string                 `json:"type"`
	} `json:"parameters,omitempty"`
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
		values = make([]string, 0)
		for _, v := range slice {
			values = append(values, v.(string))
		}
		return
	}

	condition := func(str string) string {
		switch str {
		case "string", "boolean", "number":
			return str
		default:
			if strings.HasPrefix(str, "array") {
				return "array"
			}
			return "object"
		}
	}

	// fix: type 枚举必须符合google定义，否则报错400
	// https://ai.google.dev/api/rest/v1beta/Schema?hl=zh-cn#type
	var fix func(keyv pkg.Keyv[interface{}]) (pkg.Keyv[interface{}], bool)
	{
		fix = func(properties pkg.Keyv[interface{}]) (pkg.Keyv[interface{}], bool) {
			if properties == nil {
				return nil, false
			}

			if properties.Has("type") {
				properties.Set("type", condition(properties.GetString("type")))
			}

			hasKeys := false
			for range properties {
				hasKeys = true
				break
			}

			if !hasKeys {
				// object 类型不允许空keyv
				properties.Set("properties", emp)
				return properties, false
			}

			for key := range properties {
				keyv := properties.GetKeyv(key)
				if keyv.Has("type") {
					keyv.Set("type", condition(keyv.GetString("type")))
				}
				value, _ := fix(keyv.GetKeyv("properties"))
				if value == nil {
					if keyv.Is("type", "object") {
						keyv.Set("properties", emp)
					}
				} else {
					keyv.Set("properties", value)
				}
			}

			return properties, hasKeys
		}
	}

	// 参数基本与openai对齐
	_funcDecls := make([]funcDecl, 0)
	if toolsL := len(completion.Tools); toolsL > 0 {
		for _, v := range completion.Tools {
			kv := v.GetKeyv("function").GetKeyv("parameters")
			required := kv.GetSlice("required")
			fd := funcDecl{
				// 必须为 a-z、A-Z、0-9，或包含下划线和短划线，长度上限为 63 个字符
				Name:        strings.Replace(v.GetKeyv("function").GetString("name"), "-", "_", -1),
				Description: v.GetKeyv("function").GetString("description"),
			}

			props, hasKeys := fix(kv.GetKeyv("properties"))
			if hasKeys {
				fd.Params = &struct {
					Properties map[string]interface{} `json:"properties,omitempty"`
					Required   []string               `json:"required,omitempty"`
					Type       string                 `json:"type"`
				}{
					Properties: props,
					Required:   toStrings(required),
					Type:       condition(kv.GetString("type")),
				}
			}

			_funcDecls = append(_funcDecls, fd)
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
		// tool_choice
		if tc, ok := completion.ToolChoice.(map[string]interface{}); ok {
			var toolChoice pkg.Keyv[interface{}] = tc
			if toolChoice.Is("type", "function") {
				f := toolChoice.GetKeyv("function")
				payload["toolConfig"] = map[string]interface{}{
					"functionCallingConfig": map[string]interface{}{
						"mode": "ANY",
						"allowedFunctionNames": []string{
							f.GetString("name"),
						},
					},
				}
			}
		}
	}

	marshal, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	res, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(ctx).
		POST(gURL).
		JHeader().
		Bytes(marshal).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		logger.Error(err)
		var e *url.Error
		if errors.As(err, &e) {
			e.URL = strings.Replace(e.URL, token, "AIzaSy***", -1)
		}
		if res != nil {
			h := res.Header
			if c := h.Get("content-type"); !strings.Contains(c, "text/html") {
				logger.Errorf(emit.TextResponse(res))
			}
		}
		return nil, err
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
