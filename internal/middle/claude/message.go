package claude

import (
	"bytes"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/v2/internal/common"
	"github.com/bincooo/chatgpt-adapter/v2/internal/middle"
	"github.com/bincooo/chatgpt-adapter/v2/internal/vars"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/claude-api/types"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func waitMessage(chatResponse chan types.PartialResponse, cancel func(str string) bool) (content string, err error) {

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			return "", message.Error
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

func waitResponse(ctx *gin.Context, matchers []pkg.Matcher, chatResponse chan types.PartialResponse, sse bool) {
	var (
		content = ""
		created = time.Now().Unix()
		tokens  = ctx.GetInt("tokens")
	)
	logrus.Infof("waitResponse ...")

	for {
		message, ok := <-chatResponse
		if !ok {
			break
		}

		if message.Error != nil {
			logrus.Error(message.Error)
			if middle.NotSSEHeader(ctx) {
				middle.ErrResponse(ctx, -1, message.Error)
			}
			return
		}

		fmt.Printf("----- raw -----\n %s\n", message.Text)
		raw := pkg.ExecMatchers(matchers, message.Text)
		if sse {
			middle.SSEResponse(ctx, Model, raw, created)
		}
		content += raw
	}

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		middle.Response(ctx, Model, content)
	} else {
		middle.SSEResponse(ctx, Model, "[DONE]", created)
	}
}

func mergeMessages(messages []pkg.Keyv[interface{}]) (attachment []types.Attachment, tokens int) {
	condition := func(expr string) string {
		switch expr {
		case "system", "function", "assistant":
			return expr
		case "user":
			return "human"
		default:
			return ""
		}
	}

	// 合并历史对话
	nMessages := common.MessageCombiner(messages, func(previous, next string, message map[string]string, buffer *bytes.Buffer) []string {
		role := message["role"]
		tokens += common.CalcTokens(message["content"])
		if condition(role) == condition(next) {
			// cache buffer
			if role == "function" {
				buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", message["name"], message["content"]))
				return nil
			}
			buffer.WriteString(message["content"])
			return nil
		}

		defer buffer.Reset()
		buffer.WriteString(fmt.Sprintf(message["content"]))
		return []string{
			fmt.Sprintf("%s： %s", condition(role), buffer.String()),
		}
	})

	join := strings.Join(nMessages, "\n\n")
	join = common.PadText(padMaxCount-len(join), join)

	tokens = common.CalcTokens(join)
	attachment = append(attachment, types.Attachment{
		Content:  join,
		FileName: "paste.txt",
		FileSize: len(join),
		FileType: "text/plain",
	})

	return
}
