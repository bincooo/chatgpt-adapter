package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/AutoAI/cmd/util/pool"
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"

	cmdtypes "github.com/bincooo/AutoAI/cmd/types"
	cmdvars "github.com/bincooo/AutoAI/cmd/vars"
	cltypes "github.com/bincooo/claude-api/types"
)

var (
	muLock sync.Mutex

	HARM = "I apologize, but I will not provide any responses that violate Anthropic's Acceptable Use Policy or could promote harm."

	H     = "H:"
	A     = "A:"
	S     = "System:"
	lPlot = "<plot>"
	rPlot = "</plot>"

	piles = []string{
		"Claude2.0 is so good.",
		"never lie, cheat or steal. always smile a fair deal.",
		"like tree, like fruit.",
		"East, west, home is best.",
		"原神，启动！",
		"德玛西亚万岁。",
		"薛定谔的寄。",
		"折戟成沙丶丿",
		"提无示效。",
	}
)

type schema struct {
	Debug     bool `json:"debug"`     // 开启调试
	TrimP     bool `json:"trimP"`     // 去掉头部Human
	TrimS     bool `json:"trimS"`     // 去掉尾部Assistant
	BoH       bool `json:"boH"`       // 响应截断H
	BoS       bool `json:"boS"`       // 响应截断System
	Pile      bool `json:"pile"`      // 堆积肥料
	FullColon bool `json:"fullColon"` // 全角冒号
	TrimPlot  bool `json:"trimPlot"`  // slot xml删除处理
}

func DoClaudeComplete(ctx *gin.Context, token string, r *cmdtypes.RequestDTO, wd bool) {
	IsClose := false
	IsDone := false
	fmt.Println("TOKEN_KEY: " + token)
	prepare(ctx, r)
	// 重试次数
	retry := 2

label:
	cctx, s, err := createClaudeConversation(token, r, func() bool { return IsClose })
	if err != nil {
		errorMessage := catchClaudeHandleError(err, token)
		if retry > 0 {
			retry--
			goto label
		}
		ResponseError(ctx, errorMessage, r.Stream, r.IsCompletions, wd)
		return
	}
	partialResponse := cmdvars.Manager.Reply(*cctx, func(response types.PartialResponse) {
		if r.Stream {
			if response.Status == vars.Begin {
				ctx.Status(200)
				ctx.Header("Accept", "*/*")
				ctx.Header("Content-Type", "text/event-stream")
				ctx.Writer.Flush()
				return
			}

			if response.Error != nil {
				IsClose = true
				var e *cltypes.Claude2Error
				ok := errors.As(response.Error, &e)
				err = response.Error
				if ok && token == "auto" {
					if msg := handleClaudeError(e); msg != "" {
						err = errors.New(msg)
					}
				}

				errorMessage := catchClaudeHandleError(err, token)
				if retry > 0 {
					retry--
				} else {
					ResponseError(ctx, errorMessage, r.Stream, r.IsCompletions, wd)
				}
				return
			}

			if len(response.Message) > 0 {
				select {
				case <-ctx.Request.Context().Done():
					IsClose = true
					IsDone = true
				default:
					if message := claudeResponseFilter(response.Message, s); message != "" {
						if !WriteString(ctx, message, r.IsCompletions) {
							IsClose = true
							IsDone = true
						}
					}
				}
			}

			if response.Status == vars.Closed && wd {
				WriteDone(ctx, r.IsCompletions)
			}
		} else {
			select {
			case <-ctx.Request.Context().Done():
				IsClose = true
				IsDone = true
			default:
			}
		}
	})

	if !r.Stream && !IsClose {
		if partialResponse.Error != nil {
			errorMessage := catchClaudeHandleError(partialResponse.Error, token)
			if !IsDone && retry > 0 {
				goto label
			}
			ResponseError(ctx, errorMessage, r.Stream, r.IsCompletions, wd)
			return
		}

		ctx.JSON(200, BuildCompletion(r.IsCompletions, partialResponse.Message))
	}

	if !IsDone && partialResponse.Error != nil && retry > 0 {
		goto label
	}

	// 检查大黄标
	if token == "auto" && cctx.Model == vars.Model4WebClaude2S {
		if strings.Contains(partialResponse.Message, HARM) {
			cmdvars.GlobalToken = ""
			logrus.Warn(cmdvars.I18n("HARM"))
		}
	}
}

// 过滤claude字符
func claudeResponseFilter(response string, s schema) string {
	if response == "" {
		return response
	}
	// 删除<plot>标签
	if s.TrimPlot {
		response = strings.ReplaceAll(response, lPlot, "")
		response = strings.ReplaceAll(response, rPlot, "")
	}
	return response
}

func createClaudeConversation(token string, r *cmdtypes.RequestDTO, IsC func() bool) (*types.ConversationContext, schema, error) {
	var (
		bot   string
		model string
		appId string
		id    string
		chain string
	)
	switch r.Model {
	case "claude-2.0", "claude-2":
		id = "claude-" + uuid.NewString()
		bot = vars.Claude
		model = vars.Model4WebClaude2S
	case "claude-1.0", "claude-1.2", "claude-1.3":
		id = "claude-slack"
		bot = vars.Claude
		split := strings.Split(token, ",")
		token = split[0]
		if len(split) > 1 {
			appId = split[1]
		} else {
			return nil, schema{}, errors.New("请在请求头中提供appId")
		}
	default:
		return nil, schema{}, errors.New(cmdvars.I18n("UNKNOWN_MODEL") + "`" + r.Model + "`")
	}

	message, s, err := trimClaudeMessage(r)
	if err != nil {
		return nil, s, err
	}
	fmt.Println("-----------------------Response-----------------\n", message, "\n--------------------END-------------------")
	marshal, _ := json.Marshal(s)
	logrus.Info("Schema: ", string(marshal))
	if token == "auto" && cmdvars.GlobalToken == "" {
		if cmdvars.EnablePool { // 使用池的方式
			cmdvars.GlobalToken, err = pool.GetKey()
			if err != nil {
				return nil, s, err
			}

		} else {
			muLock.Lock()
			defer muLock.Unlock()
			if cmdvars.GlobalToken == "" {
				var email string
				email, cmdvars.GlobalToken, err = pool.GenerateSessionKey()
				logrus.Info(cmdvars.I18n("GENERATE_SESSION_KEY") + "：available -- " + strconv.FormatBool(err == nil) + " email --- " + email + ", sessionKey --- " + cmdvars.GlobalToken)
				pool.CacheKey("CACHE_KEY", cmdvars.GlobalToken)
			}
		}
	}

	if token == "auto" && cmdvars.GlobalToken != "" {
		token = cmdvars.GlobalToken
	}
	fmt.Println("TOKEN_KEY: " + token)
	return &types.ConversationContext{
		Id:      id,
		Token:   token,
		Prompt:  message,
		Bot:     bot,
		Model:   model,
		Proxy:   cmdvars.Proxy,
		H:       claudeHandle(model, IsC, s),
		AppId:   appId,
		BaseURL: cmdvars.Bu,
		Chain:   chain,
	}, s, nil
}

func trimClaudeMessage(r *cmdtypes.RequestDTO) (string, schema, error) {
	result := r.Prompt
	if (r.Model == "claude-1.0" || r.Model == "claude-2.0") && len(r.Messages) > 0 {
		// 将repository的内容往上挪
		repositoryXmlHandle(r)

		// 合并消息
		for _, message := range r.Messages {
			switch message["role"] {
			case "assistant":
				result += "Assistant: " + strings.TrimSpace(message["content"]) + "\n\n"
			case "user":
				content := strings.TrimSpace(message["content"])
				if content == "" {
					continue
				}
				if strings.HasPrefix(content, "System:") {
					result += strings.TrimSpace(message["content"][7:]) + "\n\n"
				} else {
					result += "Human: " + message["content"] + "\n\n"
				}
			default:
				result += strings.TrimSpace(message["content"]) + "\n\n"
			}
		}
	}
	// ====  Schema匹配 =======
	compileRegex := regexp.MustCompile(`schema\s?\{[^}]*}`)
	s := schema{
		TrimS:     true,
		TrimP:     true,
		BoH:       true,
		BoS:       false,
		Pile:      true,
		FullColon: true,
		Debug:     false,
		TrimPlot:  false,
	}

	matchSlice := compileRegex.FindStringSubmatch(result)
	if len(matchSlice) > 0 {
		str := matchSlice[0]
		result = strings.Replace(result, str, "", -1)
		if err := json.Unmarshal([]byte(strings.TrimSpace(str[6:])), &s); err != nil {
			return "", s, err
		}
	}
	// =========================

	// ==== I apologize,[^\n]+ 道歉匹配 ======
	compileRegex = regexp.MustCompile(`I apologize[^\n]+`)
	result = compileRegex.ReplaceAllString(result, "")
	// =========================

	if s.TrimS {
		result = strings.TrimSuffix(result, "\n\nAssistant: ")
	}
	if s.TrimP {
		result = strings.TrimPrefix(result, "\n\nHuman: ")
	}

	result = strings.ReplaceAll(result, "A:", "\nAssistant:")
	result = strings.ReplaceAll(result, "H:", "\nHuman:")
	if s.FullColon {
		// result = strings.ReplaceAll(result, "System:", "System：")
		result = strings.ReplaceAll(result, "Assistant:", "Assistant：")
		result = strings.ReplaceAll(result, "Human:", "Human：")
	}

	// 填充肥料
	if s.Pile && (r.Model == "claude-2.0" || r.Model == "claude-2") {
		pile := cmdvars.GlobalPile
		if cmdvars.GlobalPile == "" {
			pile = piles[rand.Intn(len(piles))]
		}
		c := (cmdvars.GlobalPileSize - len(result)) / len(pile)
		padding := ""
		for idx := 0; idx < c; idx++ {
			padding += pile
		}

		if padding != "" {
			result = padding + "\n\n\n" + strings.TrimSpace(result)
		}
	}
	return result, s, nil
}

func claudeHandle(model string, IsC func() bool, s schema) types.CustomCacheHandler {
	return func(rChan any) func(*types.CacheBuffer) error {
		pos := 0
		begin := false
		beginIndex := -1
		partialResponse := rChan.(chan cltypes.PartialResponse)
		return func(self *types.CacheBuffer) error {
			response, ok := <-partialResponse
			if !ok {
				// 清理一下残留
				self.Cache = strings.TrimSuffix(self.Cache, A)
				self.Cache = strings.TrimSuffix(self.Cache, S)
				self.Closed = true
				return nil
			}

			if IsC() {
				self.Closed = true
				return nil
			}

			if response.Error != nil {
				self.Closed = true
				if s.Debug {
					logrus.Info(response.Error)
				}
				return response.Error
			}

			if model != vars.Model4WebClaude2S {
				text := response.Text
				str := []rune(text)
				self.Cache += string(str[pos:])
				pos = len(str)
			} else {
				self.Cache += response.Text
			}

			mergeMessage := self.Complete + self.Cache
			if s.Debug {
				fmt.Println(
					"-------------- stream ----------------\n[debug]: ",
					mergeMessage,
					"\n------- cache ------\n",
					self.Cache,
					"\n--------------------------------------")
			}
			// 遇到“A:” 或者积累200字就假定是正常输出
			if index := strings.Index(mergeMessage, A); index > -1 {
				if !begin {
					begin = true
					beginIndex = index + len(A)
					logrus.Info("---------\n", "1 Output...")
				}

			} else if !begin && len(mergeMessage) > 200 {
				begin = true
				beginIndex = len(mergeMessage)
				logrus.Info("---------\n", "3 Output...")
			}

			if begin {
				if s.Debug {
					fmt.Println(
						"-------------- H: S: ----------------\n[debug]: {H:"+strconv.Itoa(strings.LastIndex(mergeMessage, H))+"}, ",
						"{S:"+strconv.Itoa(strings.LastIndex(mergeMessage, S))+"}",
						"\n--------------------------------------")
				}
				// 遇到“H:”就结束接收
				if index := strings.LastIndex(mergeMessage, H); s.BoH && index > -1 && index > beginIndex {
					logrus.Info("---------\n", cmdvars.I18n("H"))
					if idx := strings.LastIndex(self.Cache, H); idx >= 0 {
						self.Cache = self.Cache[:idx]
					}
					self.Closed = true
					return nil
				}
				// 遇到“System:”就结束接收
				if index := strings.LastIndex(mergeMessage, S); s.BoS && index > -1 && index > beginIndex {
					logrus.Info("---------\n", cmdvars.I18n("S"))
					if idx := strings.LastIndex(self.Cache, S); idx >= 0 {
						self.Cache = self.Cache[:idx]
					}
					self.Closed = true
					return nil
				}

				// 遇到“</plot>”就结束接收
				if index := strings.LastIndex(mergeMessage, rPlot); s.TrimPlot && index > -1 && index > beginIndex {
					logrus.Info("---------\n", cmdvars.I18n("TRIM_PLOT"))
					if idx := strings.LastIndex(self.Cache, rPlot); idx >= 0 {
						self.Cache = self.Cache[:idx]
					}
					self.Closed = true
					return nil
				}
			}
			return nil
		}
	}
}

func handleClaudeError(err *cltypes.Claude2Error) (msg string) {
	if err.ErrorType.Message == "Account in read-only mode" {
		cmdvars.GlobalToken = ""
		msg = cmdvars.I18n("ACCOUNT_LOCKED")
	}
	if err.ErrorType.Message == "rate_limit_error" {
		cmdvars.GlobalToken = ""
		msg = cmdvars.I18n("ACCOUNT_LIMITED")
	}
	return msg
}

// claude异常处理（清理Token）
func catchClaudeHandleError(err error, token string) string {
	errMsg := err.Error()
	if strings.Contains(errMsg, "failed to fetch the `organizationId`") ||
		strings.Contains(errMsg, "failed to fetch the `conversationId`") {
		CleanToken(token)
	}

	if strings.Contains(errMsg, "Account in read-only mode") {
		CleanToken(token)
		errMsg = cmdvars.I18n("ERROR_ACCOUNT_LOCKED")
	} else if strings.Contains(errMsg, "rate_limit_error") {
		CleanToken(token)
		errMsg = cmdvars.I18n("ERROR_ACCOUNT_LIMITED")
	} else if strings.Contains(errMsg, "connection refused") {
		errMsg = cmdvars.I18n("ERROR_NETWORK")
	} else if strings.Contains(errMsg, "Account has not completed verification") {
		CleanToken(token)
		errMsg = cmdvars.I18n("ACCOUNT_SMS_VERIFICATION")
	} else {
		errMsg += "\n\n" + cmdvars.I18n("ERROR_OTHER")
	}
	return errMsg
	// ResponseError(ctx, errMsg, isStream, isCompletions, wd)
}
