package you

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

const ginTokens = "__tokens__"

func waitMessage(ch chan string, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-ch
		if !ok {
			break
		}

		if strings.HasPrefix(message, "error:") {
			return "", errors.New(message[6:])
		}

		if strings.HasPrefix(message, "limits:") {
			continue
		}

		logger.Debug("----- raw -----")
		logger.Debug(message)
		if len(message) > 0 {
			content += message
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, cancel chan error, ch chan string, sse bool) (content string) {
	var (
		created  = time.Now().Unix()
		tokens   = ctx.GetInt(ginTokens)
		matchers = common.GetGinMatchers(ctx)
	)

	onceExec := sync.OnceFunc(func() {
		if !sse {
			ctx.Writer.WriteHeader(http.StatusOK)
		}
	})

	logger.Info("waitResponse ...")
	for {
		select {
		case err := <-cancel:
			if err != nil {
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, err)
				}
				return
			}
			goto label
		default:
			message, ok := <-ch
			if !ok {
				raw := response.ExecMatchers(matchers, "", true)
				if raw != "" && sse {
					response.SSEResponse(ctx, Model, raw, created)
				}
				content += raw
				goto label
			}

			if strings.HasPrefix(message, "error:") {
				logger.Error(message[6:])
				if response.NotSSEHeader(ctx) {
					response.Error(ctx, -1, message[6:])
				}
				return
			}

			if strings.HasPrefix(message, "limits:") {
				continue
			}

			var raw = message
			logger.Debug("----- raw -----")
			logger.Debug(raw)
			onceExec()

			raw = response.ExecMatchers(matchers, raw, false)
			if len(raw) == 0 {
				continue
			}

			if raw == response.EOF {
				goto label
			}

			if sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
		}
	}

label:
	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	ctx.Set(vars.GinCompletionUsage, response.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}

	return
}

func mergeMessages(ctx *gin.Context, completion model.Completion) (fileMessage, chat, query string) {
	query = env.Env.GetString("you.notice")
	tokens := 0
	var (
		messages = completion.Messages
		isC      = response.IsClaude(ctx, completion.Model)
	)
	defer func() { ctx.Set(ginTokens, tokens) }()

	messageL := len(messages)
	if messageL == 1 {
		var notice = query
		message := messages[0]
		fileMessage = message.GetString("content")
		chat = message.GetString("chat")
		query = message.GetString("query")
		if notice != "" {
			query += "\n\n" + notice
		}

		join := fileMessage
		if query != "" {
			join += "\n\n" + query
		}
		if encodingLen(join) <= 12499 {
			query = join
			fileMessage = ""
		}

		tokens += response.CalcTokens(fileMessage)
		tokens += response.CalcTokens(chat)
		tokens += response.CalcTokens(query)
		return
	}

	var (
		pos      = 0
		contents []string
	)

	for {
		if pos > messageL-1 {
			break
		}

		message := messages[pos]
		convertRole, turn := response.ConvertRole(ctx, message.GetString("role"))
		if isC && message.Is("role", "system") {
			convertRole = ""
		}
		contents = append(contents, convertRole+message.GetString("content")+turn)
		pos++
	}

	convertRole, _ := response.ConvertRole(ctx, "assistant")
	fileMessage = strings.Join(contents, "") + convertRole
	tokens += response.CalcTokens(fileMessage)
	if encodingLen(fileMessage) <= 12499 {
		query = fileMessage
		fileMessage = ""
	}

	return
}

func encodingLen(str string) (count int) {
	escape := url.QueryEscape(str)
	chars := []rune(escape)
	for _, ch := range chars {
		count++
		if ch == '+' {
			count += 2
		}
	}
	return
}
