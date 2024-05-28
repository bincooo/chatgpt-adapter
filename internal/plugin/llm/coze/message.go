package coze

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/bincooo/chatgpt-adapter/internal/common"
	"github.com/bincooo/chatgpt-adapter/internal/gin.handler/response"
	"github.com/bincooo/chatgpt-adapter/internal/vars"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/coze-api"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

const ginTokens = "__tokens__"

func calcTokens(messages []coze.Message) (tokensL int) {
	for _, message := range messages {
		tokensL += common.CalcTokens(message.Content)
	}
	return
}

func waitMessage(chatResponse chan string, cancel func(str string) bool) (content string, err error) {

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
			if cancel != nil && cancel(content) {
				return content, nil
			}
		}
	}

	return content, nil
}

func waitResponse(ctx *gin.Context, matchers []common.Matcher, cancel chan error, chatResponse chan string, sse bool) (content string) {
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)

	for {
		select {
		case err := <-cancel:
			if err != nil {
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					logger.Error(err)
					response.Error(ctx, -1, err)
				}
				return
			}
			goto label
		default:
			raw, ok := <-chatResponse
			if !ok {
				goto label
			}

			if strings.HasPrefix(raw, "error: ") {
				err := strings.TrimPrefix(raw, "error: ")
				logger.Error(err)
				if response.NotSSEHeader(ctx) {
					logger.Error(err)
					response.Error(ctx, -1, err)
				}
				return
			}

			raw = strings.TrimPrefix(raw, "text: ")
			contentL := len(raw)
			if contentL <= 0 {
				continue
			}

			logger.Debug("----- raw -----")
			logger.Debug(raw)

			raw = common.ExecMatchers(matchers, raw)
			if len(raw) == 0 {
				continue
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

	ctx.Set(vars.GinCompletionUsage, common.CalcUsageTokens(content, tokens))
	if !sse {
		response.Response(ctx, Model, content)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

func mergeMessages(ctx *gin.Context) (newMessages []coze.Message, tokens int, err error) {
	var (
		proxies  = ctx.GetString("proxies")
		messages = common.GetGinCompletion(ctx).Messages
	)
	condition := func(expr string) string {
		switch expr {
		case "system", "assistant", "function", "tool", "end":
			return expr
		default:
			return "user"
		}
	}

	iterator := func(opts struct {
		Previous string
		Next     string
		Message  map[string]string
		Buffer   *bytes.Buffer
		Initial  func() pkg.Keyv[interface{}]
	}) (messages []coze.Message, _ error) {
		role := opts.Message["role"]
		tokens += common.CalcTokens(opts.Message["content"])
		// 复合消息
		if _, ok := opts.Message["multi"]; ok && role == "user" {
			message := opts.Initial()
			content, e := processMultiMessage(ctx.Request.Context(), proxies, message)
			if e != nil {
				return nil, e
			}
			opts.Buffer.WriteString(content)
			if condition(role) != condition(opts.Next) {
				messages = []coze.Message{
					{
						Role:    role,
						Content: opts.Buffer.String(),
					},
				}
				opts.Buffer.Reset()
			}
			return
		}

		if condition(role) == condition(opts.Next) {
			// cache buffer
			if role == "function" || role == "tool" {
				opts.Buffer.WriteString(fmt.Sprintf("这是系统内置tools工具的返回结果: (%s)\n\n##\n%s\n##", opts.Message["name"], opts.Message["content"]))
				return
			}
			opts.Buffer.WriteString(opts.Message["content"])
			return
		}

		defer opts.Buffer.Reset()
		opts.Buffer.WriteString(fmt.Sprintf(opts.Message["content"]))
		messages = []coze.Message{
			{
				Role:    role,
				Content: opts.Buffer.String(),
			},
		}
		return
	}

	newMessages, err = common.TextMessageCombiner(messages, iterator)
	return
}

func processMultiMessage(ctx context.Context, proxies string, message pkg.Keyv[interface{}]) (string, error) {
	contents := make([]string, 0)
	values := message.GetSlice("content")
	if len(values) == 0 {
		return "", nil
	}

	for _, value := range values {
		var keyv pkg.Keyv[interface{}]
		keyv, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		if keyv.Is("type", "text") {
			contents = append(contents, keyv.GetString("text"))
			continue
		}

		if keyv.Is("type", "image_url") {
			o := keyv.GetKeyv("image_url")
			file := o.GetString("url")
			// base64
			if strings.HasPrefix(file, "data:image/") {
				pos := strings.Index(file, ";")
				if pos == -1 {
					return "", errors.New("invalid base64 url")
				}

				mime := file[5:pos]
				ext, err := common.MimeToSuffix(mime)
				if err != nil {
					return "", err
				}

				file = file[pos+1:]
				if !strings.HasPrefix(file, "base64,") {
					return "", errors.New("invalid base64 url")
				}

				buffer := new(bytes.Buffer)
				w := multipart.NewWriter(buffer)
				fw, err := w.CreateFormFile("image", "1"+ext)
				if err != nil {
					return "", err
				}

				file, err = common.SaveBase64(file, ext[1:])
				if err != nil {
					return "", err
				}

				fileBytes, err := os.ReadFile(file)
				if err != nil {
					return "", err
				}
				_, _ = fw.Write(fileBytes)
				_ = w.Close()

				r, err := emit.ClientBuilder().
					Proxies(proxies).
					Context(ctx).
					POST("https://complete-mmx-easy-images.hf.space/upload").
					Header("Content-Type", w.FormDataContentType()).
					Header("Authorization", "Bearer 123").
					Buffer(buffer).
					DoS(http.StatusOK)
				if err != nil {
					text := emit.TextResponse(r)
					logger.Error(text)
					return "", err
				}

				obj, err := emit.ToMap(r)
				if err != nil {
					return "", err
				}

				file = obj["URL"].(string)
			}

			contents = append(contents, fmt.Sprintf("*image*: %s\n----", file))
		}
	}

	if len(contents) == 0 {
		return "", nil
	}

	return strings.Join(contents, "\n\n"), nil
}
