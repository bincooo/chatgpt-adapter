package bing

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/pkg/gpt"
	"github.com/bincooo/edge-api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
	"time"
)

const MODEL = "bing"
const sysPrompt = "This is the conversation record and description stored locally as \"JSON\" : (\" System \"is the system information,\" User \"is the user message,\" Function \"is the execution result of the built-in tool, and\" Assistant \"is the reply information of the system assistant)"

func Complete(ctx *gin.Context, req gpt.ChatCompletionRequest, matchers []common.Matcher) {
	var (
		cookie   = ctx.GetString("token")
		proxies  = ctx.GetString("proxies")
		notebook = ctx.GetBool("notebook")
	)

	options, err := edge.NewDefaultOptions(cookie, "")
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	messages := req.Messages
	messageL := len(messages)
	if messageL == 0 {
		middle.ResponseWithV(ctx, -1, "[] is too short - 'messages'")
		return
	}

	if messages[messageL-1]["role"] != "function" && len(req.Tools) > 0 {
		goOn, e := completeToolCalls(ctx, cookie, proxies, req)
		if e != nil {
			middle.ResponseWithE(ctx, -1, e)
			return
		}
		if !goOn {
			return
		}
	}

	pMessages, prompt, err := buildConversation(messages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}

	// 清理多余的标签
	matchers = appendMatchers(matchers)
	chat := edge.New(options.
		Proxies(proxies).
		TopicToE(true).
		Model(edge.ModelSydney).
		Temperature(req.Temperature))
	if notebook {
		chat.Notebook(true)
	}

	chatResponse, err := chat.Reply(ctx.Request.Context(), prompt, nil, pMessages)
	if err != nil {
		middle.ResponseWithE(ctx, -1, err)
		return
	}
	defer func() {
		go chat.Delete()
	}()
	waitResponse(ctx, matchers, chatResponse, req.Stream)
}

func appendMatchers(matchers []common.Matcher) []common.Matcher {
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
					return common.MAT_MATCHED, content
				}
				return common.MAT_MATCHING, content
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
				return common.MAT_MATCHING, content
			}
			return common.MAT_MATCHED, content
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
					return common.MAT_MATCHED, content
				}
				return common.MAT_MATCHING, content
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
				return common.MAT_MATCHING, content
			}
			return common.MAT_MATCHED, content
		},
	})
	return matchers
}

func waitMessage(chatResponse chan edge.ChatResponse) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			return "", message.Error.Message
		}

		if len(message.Text) > 0 {
			content = message.Text
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan edge.ChatResponse, sse bool) {
	var (
		pos     = 0
		content = ""
		created = time.Now().Unix()
	)

	logrus.Info("waitResponse ...")
	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			middle.ResponseWithE(ctx, -1, message.Error)
			return
		}

		var raw string
		contentL := len(message.Text)
		if pos < contentL {
			raw = message.Text[pos:contentL]
			fmt.Printf("----- raw -----\n %s\n", raw)
		}
		pos = contentL
		raw = common.ExecMatchers(matchers, raw)

		if sse {
			middle.ResponseWithSSE(ctx, MODEL, raw, created)
		} else {
			content += raw
		}
	}

	if !sse {
		middle.ResponseWith(ctx, MODEL, content)
	} else {
		middle.ResponseWithSSE(ctx, MODEL, "[DONE]", created)
	}
}

func buildConversation(messages []map[string]string) (pMessages []edge.ChatMessage, prompt string, err error) {
	pos := len(messages) - 1
	if pos < 0 {
		return
	}

	if messages[pos]["role"] == "user" {
		prompt = messages[pos]["content"]
		messages = messages[:pos]
	} else if messages[pos]["role"] == "function" {
		prompt = "继续输出"
		if pos-1 >= 0 { // 获取上一条记录
			if msg := messages[pos-1]; msg["role"] == "user" {
				prompt = msg["content"]
			}
		}
	} else {
		c := []rune(messages[pos]["content"])
		if contentL := len(c); contentL > 10 {
			prompt = fmt.Sprintf("从`%s`断点处继续写", string(c[contentL-10:]))
		} else {
			prompt = "继续输出"
		}
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

	pMessagesVar := make([]map[string]string, 0)

	// 区块
	blockProcessing := func(title string, buf []string) map[string]string {
		content := strings.Join(buf, "\n\n")
		dict := make(map[string]string)
		dict["sender"] = title
		dict["content"] = content
		return dict
	}

	// 合并历史对话
	for {
		if pos >= messageL {
			if len(buffer) > 0 {
				pMessagesVar = append(pMessagesVar, blockProcessing(strings.Title(role), buffer))
			}
			break
		}

		message := messages[pos]
		curr := condition(message["role"])
		content := message["content"]
		if curr == "" {
			return nil, "", errors.New(
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
		pMessagesVar = append(pMessagesVar, blockProcessing(strings.Title(role), buffer))
		buffer = append(make([]string, 0), content)
		role = curr
	}

	if len(pMessagesVar) > 0 {
		dict := make(map[string]interface{})
		dict["id"] = uuid.NewString()
		dict["language"] = "zh"
		dict["system_prompt"] = sysPrompt
		dict["participants"] = []string{"System", "Function", "Assistant", "User"}
		dict["messages"] = pMessagesVar
		indent, e := json.MarshalIndent(dict, "", "  ")
		if e != nil {
			return nil, "", e
		}
		pMessages = append(pMessages, edge.ChatMessage{
			"author":      "user",
			"privacy":     "Internal",
			"description": string(indent),
			"contextType": "WebPage",
			"messageType": "Context",
			"sourceName":  "history.json",
			"sourceUrl":   "file:///history.json",
			"messageId":   "discover-web--page-ping-mriduna-----",
		})
	}

	return pMessages, prompt, nil
}
