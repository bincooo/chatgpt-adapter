package grok

import (
	"bytes"
	"net/http"
	"strings"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iocgo/sdk/env"
)

var (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1.1 Safari/605.1.15"
)

type grokRequest struct {
	Temporary                 bool          `json:"temporary"`
	ModelName                 string        `json:"modelName"`
	Message                   string        `json:"message"`
	FileAttachments           []interface{} `json:"fileAttachments"`
	ImageAttachments          []interface{} `json:"imageAttachments"`
	DisableSearch             bool          `json:"disableSearch"`
	EnableImageGeneration     bool          `json:"enableImageGeneration"`
	ReturnImageBytes          bool          `json:"returnImageBytes"`
	ReturnRawGrokInXaiRequest bool          `json:"returnRawGrokInXaiRequest"`
	EnableImageStreaming      bool          `json:"enableImageStreaming"`
	ImageGenerationCount      int           `json:"imageGenerationCount"`
	ForceConcise              bool          `json:"forceConcise"`
	ToolOverrides             struct{}      `json:"toolOverrides"`
	EnableSideBySide          bool          `json:"enableSideBySide"`
	IsPreset                  bool          `json:"isPreset"`
	SendFinalMetadata         bool          `json:"sendFinalMetadata"`
	CustomInstructions        string        `json:"customInstructions"`
	DeepsearchPreset          string        `json:"deepsearchPreset"`
	IsReasoning               bool          `json:"isReasoning"`
}

func fetch(ctx *gin.Context, proxied, cookie string, request grokRequest) (response *http.Response, err error) {
	ua := ctx.GetString("userAgent")
	lang := ctx.GetString("lang")
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(proxied).
		POST("https://grok.com/rest/app-chat/conversations/new").
		JSONHeader().
		Header("origin", "https://grok.com").
		Header("referer", "https://grok.com/").
		Header("baggage", "sentry-environment=production,sentry-release="+common.Hex(21)+",sentry-public_key="+strings.ReplaceAll(uuid.NewString(), "-", "")+",sentry-trace_id="+strings.ReplaceAll(uuid.NewString(), "-", "")+",sentry-replay_id="+strings.ReplaceAll(uuid.NewString(), "-", "")+",sentry-sample_rate=1,sentry-sampled=true").
		Header("sentry-trace", strings.ReplaceAll(uuid.NewString(), "-", "")+"-"+common.Hex(16)+"-1").
		Header("accept-language", elseOf(lang == "", "en-US,en;q=0.9", lang)).
		Header("user-agent", elseOf(ua == "", userAgent, ua)).
		Header("cookie", emit.MergeCookies(cookie, ctx.GetString("clearance"))).
		Body(request).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	return
}

func convertRequest(ctx *gin.Context, env *env.Environment, completion model.Completion) (request grokRequest, err error) {
	contentBuffer := new(bytes.Buffer)
	customInstructions := ""

	if len(completion.Messages) == 1 {
		contentBuffer.WriteString(completion.Messages[0].GetString("content"))
		goto label
	}

	for idx, message := range completion.Messages {
		if idx == 0 && message.Is("role", "system") {
			customInstructions = message.GetString("content")
			continue
		}
		role, end := response.ConvertRole(ctx, message.GetString("role"))
		contentBuffer.WriteString(role)
		contentBuffer.WriteString(message.GetString("content"))
		contentBuffer.WriteString(end)
	}

label:
	request = grokRequest{
		Temporary:                 true,
		ModelName:                 completion.Model,
		FileAttachments:           make([]interface{}, 0),
		ImageAttachments:          make([]interface{}, 0),
		EnableImageGeneration:     true,
		ReturnImageBytes:          false,
		ReturnRawGrokInXaiRequest: false,
		EnableImageStreaming:      true,
		ImageGenerationCount:      1,
		ForceConcise:              false,
		ToolOverrides:             struct{}{},
		EnableSideBySide:          true,
		IsPreset:                  false,
		SendFinalMetadata:         true,
		DeepsearchPreset:          "",
		CustomInstructions:        customInstructions,
		Message:                   contentBuffer.String(),
		DisableSearch:             env.GetBool("grok.disable_search"),
		IsReasoning:               env.GetBool("grok.think_reason"),
	}
	return
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}
