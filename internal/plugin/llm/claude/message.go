package claude

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	claude3 "github.com/bincooo/claude-api"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func waitMessage(chatResponse chan claude3.PartialResponse, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			return "", logger.WarpError(message.Error)
		}

		if len(message.Text) > 0 {
			content += message.Text
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, chatResponse chan claude3.PartialResponse, sse bool) (content string) {
	var (
		created = time.Now().Unix()
		tokens  = ctx.GetInt(ginTokens)
	)
	logger.Infof("waitResponse ...")

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			logger.Error(message.Error)
			if response.NotSSEHeader(ctx) {
				logger.Error(message.Error)
				response.Error(ctx, -1, message.Error)
			}
			return
		}

		logger.Debug("----- raw -----")
		logger.Debug(message.Text)

		raw := common.ExecMatchers(matchers, message.Text)
		if len(raw) == 0 {
			continue
		}

		if sse {
			response.SSEResponse(ctx, Model, raw, created)
		}
		content += raw
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

func mergeMessages(ctx *gin.Context) (attachment []claude3.Attachment, tokens int) {
	messages, _ := common.GetGinValue[[]pkg.Keyv[interface{}]](ctx, vars.GinClaudeMessages)
	var contents []string
	for _, message := range messages {
		contents = append(contents, message.GetString("content"))
	}

	join := strings.Join(contents, "\n\n")
	if ctx.GetBool("pad") {
		count := ctx.GetInt("claude.pad")
		if count == 0 {
			count = padMaxCount
		}
		join = common.PadJunkMessage(count-len(join), join)
	}

	tokens = common.CalcTokens(join)
	attachment = append(attachment, claude3.Attachment{
		Content:  join,
		FileName: "paste.txt",
		FileSize: len(join),
		FileType: "text/plain",
	})

	return
}
