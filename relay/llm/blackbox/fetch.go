package blackbox

import (
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"context"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"net/http"
)

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1.1 Safari/605.1.15"
)

type blackboxRequest struct {
	Messages []model.Keyv[interface{}] `json:"messages"`

	AgentMode         map[string]interface{} `json:"agentMode"`
	TrendingAgentMode map[string]interface{} `json:"trendingAgentMode"`

	Id                    string  `json:"id"`
	CodeModelMode         bool    `json:"codeModelMode"`
	IsMicMode             bool    `json:"isMicMode"`
	UserSystemPrompt      string  `json:"userSystemPrompt"`
	MaxTokens             int     `json:"maxTokens"`
	PlaygroundTopP        float32 `json:"playgroundTopP"`
	PlaygroundTemperature float32 `json:"playgroundTemperature"`
	IsChromeExt           bool    `json:"isChromeExt"`
	GithubToken           string  `json:"githubToken"`
	ClickedAnswer2        bool    `json:"clickedAnswer2"`
	ClickedAnswer3        bool    `json:"clickedAnswer3"`
	ClickedForceWebSearch bool    `json:"clickedForceWebSearch"`
	VisitFromDelta        bool    `json:"visitFromDelta"`
	MobileClient          bool    `json:"mobileClient"`
	UserSelectedModel     string  `json:"userSelectedModel"`
	Validated             string  `json:"validated"`
	ImageGenerationMode   bool    `json:"imageGenerationMode"`
	WebSearchModePrompt   bool    `json:"webSearchModePrompt"`
}

func fetch(ctx context.Context, proxied string, cookie string, request blackboxRequest) (response *http.Response, err error) {
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST("https://www.blackbox.ai/api/chat").
		JHeader().
		Header("Cookie", "sessionId="+cookie).
		Header("origin", "https://www.blackbox.ai").
		Header("referer", "https://www.blackbox.ai/").
		Header("user-agent", userAgent).
		Body(request).
		DoC(emit.Status(http.StatusOK), emit.IsTEXT)
	return
}

func convertRequest(ctx *gin.Context, env *env.Environment, completion model.Completion) (request blackboxRequest) {
	request.Messages = completion.Messages
	if response.IsClaude(ctx, completion.Model) {
		request.Messages = completion.Messages[:1]
		request.Messages[0].Set("role", "user")
	}

	id := request.Messages[0].GetString("id")
	if id == "" {
		id = common.Hex(7)
	}

	request.Id = id
	request.MaxTokens = completion.MaxTokens
	request.PlaygroundTopP = completion.TopP
	request.PlaygroundTemperature = completion.Temperature
	request.UserSelectedModel = completion.Model[9:]
	request.Validated = env.GetString("blackbox")
	request.AgentMode = make(map[string]interface{})
	request.TrendingAgentMode = make(map[string]interface{})
	return
}
