package qodo

import (
	"bufio"
	"chatgpt-adapter/core/gin/inter"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

const (
	ginTokens = "__tokens__"
	b         = "*"
)

type qodoResponse struct {
	Type      string `json:"type"`
	SubType   string `json:"sub_type"`
	UsedModel string `json:"used_model"`
	Data      struct {
		Title              string      `json:"title"`
		Content            string      `json:"content"`
		CanBeRepliedTo     bool        `json:"can_be_replied_to"`
		IsStreaming        bool        `json:"is_streaming"`
		ArtifactAttributes interface{} `json:"artifact_attributes"`
	} `json:"data"`
}

func waitMessage(r *http.Response, cancel func(str string) bool) (content string, err error) {
	defer r.Body.Close()
	reader := bufio.NewReader(r.Body)
	var dataBytes []byte
	for {
		dataBytes, _, err = reader.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return
		}

		var res qodoResponse
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		delta := res.Data
		if delta.Content == "" {
			continue
		}

		raw := delta.Content
		logger.Debug("----- raw -----")
		logger.Debug(raw)
		content += raw
		if cancel != nil && cancel(content) {
			return content, nil
		}
	}
	return
}

func waitResponse(ctx *gin.Context, r *http.Response, sse bool) (content string) {
	created := time.Now().Unix()
	logger.Infof("waitResponse ...")
	tokens := ctx.GetInt(ginTokens)
	thinkReason := env.Env.GetBool("server.think_reason")
	reasoningContent := ""

	onceExec := sync.OnceFunc(func() {
		if !sse {
			ctx.Writer.WriteHeader(http.StatusOK)
		}
	})

	var (
		matchers = common.GetGinMatchers(ctx)
	)

	matchers = addUnpackMatcher(env.Env, matchers)

	defer r.Body.Close()
	reader := bufio.NewReader(r.Body)
	think := 0
	for {
		dataBytes, _, err := reader.ReadLine()
		if err == io.EOF {
			raw := response.ExecMatchers(matchers, "", true)
			if raw != "" && sse {
				response.SSEResponse(ctx, Model, raw, created)
			}
			content += raw
			break
		}

		if asError(ctx, err) {
			return
		}

		var res qodoResponse
		if len(dataBytes) == 0 {
			continue
		}

		err = json.Unmarshal(dataBytes, &res)
		if err != nil {
			logger.Warn(err)
			continue
		}

		delta := res.Data
		reasonContent := ""

		raw := delta.Content
		if thinkReason && think == 0 {
			if strings.HasPrefix(raw, "<think>") {
				reasonContent = raw[7:]
				raw = ""
				think = 1
			}
		}

		if thinkReason && think == 1 {
			reasonContent = raw
			if strings.HasPrefix(raw, "</think>") {
				reasonContent = ""
				think = 2
			}

			raw = ""
			logger.Debug("----- think raw -----")
			logger.Debug(reasonContent)
			reasoningContent += reasonContent
			goto label
		}

		logger.Debug("----- raw -----")
		logger.Debug(raw)
		onceExec()

		raw = response.ExecMatchers(matchers, raw, false)
		if len(raw) == 0 {
			continue
		}

	label:
		if raw == response.EOF {
			break
		}

		if sse {
			response.ReasonSSEResponse(ctx, Model, raw, reasonContent, created)
		}
		content += raw
	}

	if content == "" && response.NotSSEHeader(ctx) {
		return
	}
	ctx.Set(vars.GinCompletionUsage, response.CalcUsageTokens(reasoningContent+content, tokens))
	if !sse {
		response.ReasonResponse(ctx, Model, content, reasoningContent)
	} else {
		response.SSEResponse(ctx, Model, "[DONE]", created)
	}
	return
}

// 还原字面转义
func addUnpackMatcher(env *env.Environment, matchers []inter.Matcher) []inter.Matcher {
	maxLen := 5
	return append(matchers, response.NewMatcher(b, func(index int, content string) (state int, cache, result string) {
		rc := []rune(content)
		if index+maxLen > len(rc)-1 {
			return response.MatMatching, "", content
		}
		logger.Infof("execute matcher[<b>] content:\n%s", content)
		for k, v := range mapC {
			content = strings.ReplaceAll(content, b+v+b, k)
		}
		mapCc := env.GetStringMapString("qodo.mapC")
		for k, v := range mapCc {
			content = strings.ReplaceAll(content, b+v+b, k)
		}
		return response.MatMatched, cache, content
	}))
}

func asError(ctx *gin.Context, err error) (ok bool) {
	if err == nil {
		return
	}

	logger.Error(err)
	if response.NotSSEHeader(ctx) {
		response.Error(ctx, -1, err)
	}
	ok = true
	return
}
