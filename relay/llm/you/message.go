package you

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/toolcall"
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
		messages    = completion.Messages
		specialized = ctx.GetBool("specialized")
		isC         = response.IsClaude(ctx, "", completion.Model)
	)
	defer func() { ctx.Set(ginTokens, tokens) }()

	messageL := len(messages)

	if specialized && isC && messageL == 3 {
		var notice = query
		fileMessage = messages[0].GetString("content")
		chat = messages[1].GetString("content")
		query = messages[2].GetString("content")
		if notice != "" {
			query += "\n\n" + notice
		}
		tokens += response.CalcTokens(fileMessage)
		tokens += response.CalcTokens(chat)
		tokens += response.CalcTokens(query)
		return
	}

	if messageL == 1 {
		message := messages[0]
		content := message.GetString("content")
		if len([]rune(content)) < 2500 {
			query = content
		} else {
			fileMessage = content
		}
		tokens += response.CalcTokens(fileMessage)
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
	return
}

func echoMessages(ctx *gin.Context, fileMessage, chat, message string) {
	var (
		completion   = common.GetGinCompletion(ctx)
		toolMessages = toolcall.ExtractToolMessages(&completion)
	)

	content := ""
	if len(toolMessages) > 0 {
		content += "\n----------toolCallMessages----------\n"
		chunkBytes, _ := json.MarshalIndent(toolMessages, "", "  ")
		content += string(chunkBytes)
	}

	response.Echo(ctx, completion.Model, fmt.Sprintf(
		"--------FILE MESSAGE--------:\n%s\n\n--------CHAT MESSAGE--------:\n%s\n\n--------CURR QUESTION--------:\n%s",
		fileMessage,
		chat,
		message,
	)+content, completion.Stream)
}
