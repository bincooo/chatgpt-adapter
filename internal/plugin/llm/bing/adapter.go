package bing

import (
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/plugin"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
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
	return Model == model
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      Model,
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
		pad      = ctx.GetBool("pad")

		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	options, err := edge.NewDefaultOptions(cookie, "")
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

	maxCount := 8
	if chat.IsLogin() {
		maxCount = 28
	}

	pMessages, currMessage, tokens := mergeMessages(pad, maxCount, completion.Messages)

	// 清理多余的标签
	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	ctx.Set(ginTokens, tokens)

	r, err := chat.Reply(ctx.Request.Context(), currMessage, pMessages)
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
	if content == "" && response.NotSSEHeader(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func joinMatchers(ctx *gin.Context, matchers []common.Matcher) (chan error, []common.Matcher) {
	// 清理 [1]、[2] 标签
	// 清理 [^1^]、[^2^] 标签
	// 清理 [^1^ 标签
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "[",
		H: func(index int, content string) (state int, result string) {
			r := []rune(content)
			eIndex := len(r) - 1
			if index+4 > eIndex {
				if index <= eIndex && r[index] != []rune("^")[0] {
					return vars.MatMatched, content
				}
				return vars.MatMatching, content
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
				return vars.MatMatching, content
			}
			return vars.MatMatched, content
		},
	})
	// (^1^) (^1^ (^1^^ 标签
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "(",
		H: func(index int, content string) (state int, result string) {
			r := []rune(content)
			eIndex := len(r) - 1
			if index+4 > eIndex {
				if index <= eIndex && r[index] != []rune("^")[0] {
					return vars.MatMatched, content
				}
				return vars.MatMatching, content
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
				return vars.MatMatching, content
			}
			return vars.MatMatched, content
		},
	})

	// 自定义标记块中断
	cancel, matcher := common.NewCancelMather(ctx)
	matchers = append(matchers, matcher)
	return cancel, matchers
}
