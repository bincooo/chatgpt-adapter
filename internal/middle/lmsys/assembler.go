package lmsys

import (
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const (
	MODEL = "lmsys"
)

var (
	blocks = []string{
		"<|system|>",
		"<|user|>",
		"<|assistant|>",
		"<|function|>",
		"<|end|>",
	}
)

func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []common.Matcher) {
	req.Model = req.Model[6:]
	proxies := ctx.GetString("proxies")
	messages := req.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, -1, "[] is too short - 'messages'")
		return
	}

	if messages[messageL-1]["role"] != "function" && len(req.Tools) > 0 {
		goOn, e := completeToolCalls(ctx, proxies, req)
		if e != nil {
			errMessage := e.Error()
			if strings.Contains(errMessage, "Login verification is invalid") {
				middle.ResponseWithV(ctx, http.StatusUnauthorized, errMessage)
			}
			middle.ResponseWithV(ctx, -1, errMessage)
			return
		}
		if !goOn {
			return
		}
	}

	newMessages, tokens, err := buildConversation(messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	cancel := make(chan error, 1)
	ctx.Set("tokens", tokens)

	ch, err := fetch(ctx, proxies, newMessages, options{
		model:       req.Model,
		temperature: req.Temperature,
		topP:        req.TopP,
		maxTokens:   req.MaxTokens,
	})
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	// 自定义标记块中断
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "<|",
		H: func(index int, content string) (state int, result string) {
			if len(content) < 13 {
				return common.MAT_MATCHING, content
			}

			for _, block := range blocks {
				if strings.Contains(content, block) {
					cancel <- nil
					return common.MAT_MATCHED, ""
				}
			}
			return common.MAT_DEFAULT, content
		},
	})

	// 违反内容中断并返回错误
	matchers = append(matchers, &common.SymbolMatcher{
		Find: "I did not actually provide any input that could violate",
		H: func(index int, content string) (state int, result string) {
			cancel <- errors.New("I did not actually provide any input that could violate content guidelines")
			return common.MAT_MATCHED, ""
		},
	})

	waitResponse(ctx, matchers, ch, cancel, req.Stream)
}

func waitMessage(chatResponse chan string) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if strings.HasPrefix(message, "error: ") {
			return "", errors.New(strings.TrimPrefix(message, "error: "))
		}

		message = strings.TrimPrefix(message, "text: ")
		if len(message) > 0 {
			content += message
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan string, cancel chan error, sse bool) {
	content := ""
	created := time.Now().Unix()
	logrus.Infof("waitResponse ...")
	tokens := ctx.GetInt("tokens")

	for {
		select {
		case err := <-cancel:
			if err != nil {
				middle.ResponseWithE(ctx, -1, err)
				return
			}
			goto label
		default:
			raw, ok := <-chatResponse
			if !ok {
				goto label
			}

			if strings.HasPrefix(raw, "error: ") {
				middle.ResponseWithV(ctx, -1, strings.TrimPrefix(raw, "error: "))
				return
			}

			raw = strings.TrimPrefix(raw, "text: ")
			contentL := len(raw)
			if contentL <= 0 {
				continue
			}

			fmt.Printf("----- raw -----\n %s\n", raw)
			raw = common.ExecMatchers(matchers, raw)
			if sse && len(raw) > 0 {
				middle.ResponseWithSSE(ctx, MODEL, raw, nil, created)
			}
			content += raw
		}
	}

label:
	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", common.CalcUsageTokens(content, tokens), created)
	}
}

func buildConversation(messages []map[string]string) (newMessages string, tokens int, err error) {
	var prompt string
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	if messages[pos]["role"] == "function" {
		prompt = "继续输出"
		if pos-1 >= 0 { // 获取上一条记录
			if msg := messages[pos-1]; msg["role"] == "user" {
				prompt = msg["content"]
			}
		}
	} else if messages[pos]["role"] != "user" {
		c := []rune(messages[pos]["content"])
		if contentL := len(c); contentL > 10 {
			prompt = fmt.Sprintf("从`%s`断点处继续写", string(c[contentL-10:]))
		} else {
			prompt = "继续输出"
		}
	}

	if len(prompt) > 0 {
		messages = append(messages, map[string]string{
			"role":    "user",
			"content": prompt,
		})
	}

	pos = 0
	messageL := len(messages)

	role := ""
	buffer := make([]string, 0)

	condition := func(expr string) string {
		switch expr {
		case "system", "user", "function", "assistant":
			return expr
		default:
			return ""
		}
	}

	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				tokens += common.CalcTokens(strings.Join(buffer, ""))
				newMessages += fmt.Sprintf("<|%s|>\n%s<|end|>", role, strings.Join(buffer, "\n\n"))
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return "", -1, errors.New(
				fmt.Sprintf("'%s' is not one of ['system', 'assistant', 'user', 'function'] - 'messages.%d.role'",
					message["role"], pos))
		}

		pos++
		if role == "" {
			role = curr
		}

		if curr == "function" {
			content = fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], content)
		}

		if curr == role {
			buffer = append(buffer, content)
			continue
		}

		tokens += common.CalcTokens(strings.Join(buffer, ""))
		newMessages += fmt.Sprintf("<|%s|>\n%s<|end|>\n\n", role, strings.Join(buffer, "\n\n"))
		buffer = append(make([]string, 0), content)
		role = curr
	}

	newMessages += "\n<|assistant|>"
	return newMessages, tokens, nil
}
