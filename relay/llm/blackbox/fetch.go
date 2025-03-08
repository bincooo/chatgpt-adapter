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

	Id            string      `json:"id"`
	PreviewToken  interface{} `json:"previewToken"`
	UserId        interface{} `json:"userId"`
	CodeModelMode bool        `json:"codeModelMode"`
	AgentMode     struct {
	} `json:"agentMode"`
	TrendingAgentMode struct {
	} `json:"trendingAgentMode"`
	IsMicMode             bool        `json:"isMicMode"`
	MaxTokens             int         `json:"maxTokens"`
	PlaygroundTopP        interface{} `json:"playgroundTopP"`
	PlaygroundTemperature interface{} `json:"playgroundTemperature"`
	IsChromeExt           bool        `json:"isChromeExt"`
	GithubToken           string      `json:"githubToken"`
	ClickedAnswer2        bool        `json:"clickedAnswer2"`
	ClickedAnswer3        bool        `json:"clickedAnswer3"`
	ClickedForceWebSearch bool        `json:"clickedForceWebSearch"`
	VisitFromDelta        bool        `json:"visitFromDelta"`
	MobileClient          bool        `json:"mobileClient"`
	UserSelectedModel     string      `json:"userSelectedModel"`
	Validated             string      `json:"validated"`
	ImageGenerationMode   bool        `json:"imageGenerationMode"`
	WebSearchModePrompt   bool        `json:"webSearchModePrompt"`
}

func fetch(ctx context.Context, proxied, cookie string, request blackboxRequest) (response *http.Response, err error) {
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST("https://www.blackbox.ai/api/chat").
		JSONHeader().
		Header("accept-language", "en-US,en;q=0.9").
		Header("origin", "https://www.blackbox.ai").
		Header("referer", "https://www.blackbox.ai/").
		Header("user-agent", userAgent).
		Header("cookie", cookie).
		Body(request).
		DoC(emit.Status(http.StatusOK), emit.IsTEXT)
	return
}

func convertRequest(ctx *gin.Context, env *env.Environment, completion model.Completion) (request blackboxRequest) {
	request.Messages = completion.Messages
	specialized := ctx.GetBool("specialized")
	if specialized && response.IsClaude(ctx, completion.Model) {
		request.Messages = completion.Messages[:1]
		request.Messages[0].Set("role", "user")
	}

	id := request.Messages[0].GetString("id")
	if id == "" {
		id = common.Hex(7)
		request.Messages[0].Set("id", id)
	}

	request.Id = id
	request.CodeModelMode = true
	request.MaxTokens = completion.MaxTokens
	request.PlaygroundTopP = completion.TopP
	request.PlaygroundTemperature = completion.Temperature
	request.UserSelectedModel = completion.Model[9:]
	request.Validated = env.GetString("blackbox.token")
	request.AgentMode = struct{}{}
	request.TrendingAgentMode = struct{}{}
	return
}
