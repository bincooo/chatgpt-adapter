package bing

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"fmt"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"regexp"
	"strings"
)

var (
	Adapter = API{}
	Model   = "bing"
)

type API struct {
	plugin.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	switch model {
	case Model, Model + "-online", Model + "-vision":
		return true
	default:
		return false
	}
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      Model,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      Model + "-online",
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      Model + "-vision",
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		cookie   = ctx.GetString("token")
		proxies  = ctx.GetString("proxies")
		notebook = ctx.GetBool("notebook")

		pad  = ctx.GetBool("pad")
		echo = ctx.GetBool(vars.GinEcho)

		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)

		baseUrl = pkg.Config.GetString("bing.base-url")
	)

	if cookie == "xxx" {
		cookie = common.RandString(32)
	}

	if completion.Model == Model+"-vision" {
		pad = true
	}

	options, err := edge.NewDefaultOptions(cookie, baseUrl)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	chat := edge.New(options.
		Proxies(proxies).
		TopicToE(true).
		Model(edge.ModelSydney).
		Temperature(completion.Temperature))
	if notebook {
		chat.Notebook(true)
	}

	chat.Client(plugin.HTTPClient)
	if completion.Model == "bing-online" {
		chat.Plugins(edge.PluginSearch)
	} else {
		chat.JoinOptionSets(edge.OptionSets_Nosearchall)
	}

	maxCount := 2
	if chat.IsLogin(common.GetGinContext(ctx)) {
		maxCount = 28
	}

	pMessages, currMessage, tokens, err := mergeMessages(ctx, pad, maxCount, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if echo {
		bytes, _ := json.MarshalIndent(pMessages, "", "  ")
		response.Echo(ctx, completion.Model, fmt.Sprintf("PREVIOUS MESSAGES:\n%s\n\n\n------\nCURR QUESTION:\n%s", bytes, currMessage), completion.Stream)
		return
	}

	// 清理多余的标签
	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	ctx.Set(ginTokens, tokens)

	r, err := chat.Reply(common.GetGinContext(ctx), currMessage, pMessages)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	slice := strings.Split(chat.GetSession().ConversationId, "|")
	if len(slice) > 1 {
		logger.Infof("bing status: [%s]", slice[1])
	}

	content := waitResponse(ctx, matchers, cancel, r, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func joinMatchers(ctx *gin.Context, matchers []common.Matcher) (chan error, []common.Matcher) {
	// 清理 [1]、[2] 标签
	// 清理 [^1^]、[^2^] 标签
	// 清理 [^1^ 标签
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "[",
		H: func(index int, content string) (state int, _, result string) {
			r := []rune(content)
			eIndex := len(r) - 1
			if index+4 > eIndex {
				if index <= eIndex && r[index] != []rune("^")[0] {
					return vars.MatMatched, "", content
				}
				return vars.MatMatching, "", content
			}
			regexCompile := regexp.MustCompile(`\[\d+]`)
			content = regexCompile.ReplaceAllString(content, "")
			regexCompile = regexp.MustCompile(`\[\^\d+\^]:`)
			content = regexCompile.ReplaceAllString(content, "")
			regexCompile = regexp.MustCompile(`\[\^\d+\^]`)
			content = regexCompile.ReplaceAllString(content, "")
			regexCompile = regexp.MustCompile(`\[\^\d+\^\^`)
			content = regexCompile.ReplaceAllString(content, "")
			regexCompile = regexp.MustCompile(`\[\^\d+\^`)
			content = regexCompile.ReplaceAllString(content, "")
			if strings.HasSuffix(content, "[") || strings.HasSuffix(content, "[^") {
				return vars.MatMatching, "", content
			}
			return vars.MatMatched, "", content
		},
	})
	// (^1^) (^1^ (^1^^ 标签
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "(",
		H: func(index int, content string) (state int, _, result string) {
			r := []rune(content)
			eIndex := len(r) - 1
			if index+4 > eIndex {
				if index <= eIndex && r[index] != []rune("^")[0] {
					return vars.MatMatched, "", content
				}
				return vars.MatMatching, "", content
			}
			regexCompile := regexp.MustCompile(`\(\^\d+\^\):`)
			content = regexCompile.ReplaceAllString(content, "")
			regexCompile = regexp.MustCompile(`\(\^\d+\^\)`)
			content = regexCompile.ReplaceAllString(content, "")
			regexCompile = regexp.MustCompile(`\(\^\d+\^\^`)
			content = regexCompile.ReplaceAllString(content, "")
			regexCompile = regexp.MustCompile(`\(\^\d+\^`)
			content = regexCompile.ReplaceAllString(content, "")
			if strings.HasSuffix(content, "(") || strings.HasSuffix(content, "(^") {
				return vars.MatMatching, "", content
			}
			return vars.MatMatched, "", content
		},
	})

	// 自定义标记块中断
	cancel, matcher := common.NewCancelMatcher(ctx)
	matchers = append(matchers, matcher...)
	return cancel, matchers
}
