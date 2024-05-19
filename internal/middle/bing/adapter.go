package bing

import (
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

var (
	Adapter = API{}
	Model   = "bing"
)

type API struct {
	middle.BaseAdapter
}

func (API) Match(_ *gin.Context, model string) bool {
	return Model == model
}

func (API) Models() []middle.Model {
	return []middle.Model{
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
		middle.ErrResponse(ctx, -1, err)
		return
	}

	if common.NeedToToolCall(ctx) {
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
	cancel, matchers = addMatchers(matchers)
	ctx.Set("tokens", tokens)
	chatResponse, err := chat.Reply(ctx.Request.Context(), currMessage, pMessages)
	if err != nil {
		middle.ErrResponse(ctx, -1, err)
		return
	}

	slices := strings.Split(chat.GetSession().ConversationId, "|")
	if len(slices) > 1 {
		logrus.Infof("bing status: [%s]", slices[1])
	}
	waitResponse(ctx, matchers, cancel, chatResponse, completion.Stream)
}

func addMatchers(matchers []pkg.Matcher) (chan error, []pkg.Matcher) {
	// 清理 [1]、[2] 标签
	// 清理 [^1^]、[^2^] 标签
	// 清理 [^1^ 标签
	matchers = append(matchers, &pkg.SymbolMatcher{
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
	matchers = append(matchers, &pkg.SymbolMatcher{
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
	cancel, matcher := pkg.NewCancelMather()
	matchers = append(matchers, matcher)
	return cancel, matchers
}
