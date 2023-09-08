package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bincooo/AutoAI/types"
	"github.com/bincooo/AutoAI/vars"
	"github.com/bincooo/claude-api/util"
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
	clTypes "github.com/bincooo/claude-api/types"
)

var (
	muLock sync.Mutex

	HARM = "I apologize, but I will not provide any responses that violate Anthropic's Acceptable Use Policy or could promote harm."

	H = "H:"
	A = "A:"
	S = "System:"

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
}

func DoClaudeComplete(ctx *gin.Context, token string, r *cmdtypes.RequestDTO) {
	IsClose := false
	context, err := createClaudeConversation(token, r, func() bool { return IsClose })
	if err != nil {
		responseClaudeError(ctx, err, r.Stream, r.IsCompletions, token)
		return
	}
	partialResponse := cmdvars.Manager.Reply(*context, func(response types.PartialResponse) {
		if r.Stream {
			if response.Status == vars.Begin {
				ctx.Status(200)
				ctx.Header("Accept", "*/*")
				ctx.Header("Content-Type", "text/event-stream")
				ctx.Writer.Flush()
				return
			}

			if response.Error != nil {
				var e *clTypes.Claude2Error
				ok := errors.As(response.Error, &e)
				err = response.Error
				if ok && token == "auto" {
					if msg := handleClaudeError(e); msg != "" {
						err = errors.New(msg)
					}
				}

				responseClaudeError(ctx, err, r.Stream, r.IsCompletions, token)
				return
			}

			if len(response.Message) > 0 {
				select {
				case <-ctx.Request.Context().Done():
					IsClose = true
				default:
					if !WriteString(ctx, response.Message, r.IsCompletions) {
						IsClose = true
					}
				}
			}

			if response.Status == vars.Closed {
				WriteDone(ctx, r.IsCompletions)
			}
		} else {
			select {
			case <-ctx.Request.Context().Done():
				IsClose = true
			default:
			}
		}
	})

	if !r.Stream && !IsClose {
		if partialResponse.Error != nil {
			responseClaudeError(ctx, partialResponse.Error, r.Stream, r.IsCompletions, token)
			return
		}

		ctx.JSON(200, BuildCompletion(r.IsCompletions, partialResponse.Message))
	}

	// 检查大黄标
	if token == "auto" && context.Model == vars.Model4WebClaude2S {
		if strings.Contains(partialResponse.Message, HARM) {
			cmdvars.GlobalToken = ""
			logrus.Warn(cmdvars.I18n("HARM"))
		}
	}
}

func createClaudeConversation(token string, r *cmdtypes.RequestDTO, IsC func() bool) (*types.ConversationContext, error) {
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
			return nil, errors.New("请在请求头中提供appId")
		}
	default:
		return nil, errors.New(cmdvars.I18n("UNKNOWN_MODEL") + "`" + r.Model + "`")
	}

	message, s, err := trimClaudeMessage(r)
	if err != nil {
		return nil, err
	}
	fmt.Println("-----------------------Response-----------------\n", message, "\n--------------------END-------------------")
	logrus.Info("Schema: ", s)
	if token == "auto" && cmdvars.GlobalToken == "" {
		muLock.Lock()
		defer muLock.Unlock()
		if cmdvars.GlobalToken == "" {
			var email string
			email, cmdvars.GlobalToken, err = util.LoginFor(cmdvars.Bu, cmdvars.Suffix, cmdvars.Proxy)
			if err != nil {
				logrus.Error(cmdvars.I18n("FAILED_GENERATE_SESSION_KEY")+"： email ---"+email, err)
				return nil, err
			}
			logrus.Info(cmdvars.I18n("GENERATE_SESSION_KEY") + "： email --- " + email + ", sessionKey --- " + cmdvars.GlobalToken)
			CacheKey("CACHE_KEY", cmdvars.GlobalToken)
		}
	}

	if token == "auto" && cmdvars.GlobalToken != "" {
		token = cmdvars.GlobalToken
	}

	return &types.ConversationContext{
		Id:      id,
		Token:   token,
		Prompt:  message,
		Bot:     bot,
		Model:   model,
		Proxy:   cmdvars.Proxy,
		H:       claudeHandle(model, IsC, s.BoH, s.BoS, s.Debug),
		AppId:   appId,
		BaseURL: cmdvars.Bu,
		Chain:   chain,
	}, nil
}

func trimClaudeMessage(r *cmdtypes.RequestDTO) (string, schema, error) {
	result := r.Prompt
	if (r.Model == "claude-1.0" || r.Model == "claude-2.0") && len(r.Messages) > 0 {
		for _, message := range r.Messages {
			switch message["role"] {
			case "assistant":
				result += "Assistant: " + message["content"] + "\n\n"
			case "user":
				result += "Human: " + message["content"] + "\n\n"
			default:
				result += message["content"] + "\n\n"
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
	}

	matchSlice := compileRegex.FindStringSubmatch(r.Prompt)
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

func claudeHandle(model string, IsC func() bool, boH, boS, debug bool) func(rChan any) func(*types.CacheBuffer) error {
	return func(rChan any) func(*types.CacheBuffer) error {
		pos := 0
		begin := false
		beginIndex := -1
		partialResponse := rChan.(chan clTypes.PartialResponse)
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
				if debug {
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
			if debug {
				logrus.Info(
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
					beginIndex = index
					logrus.Info("---------\n", "1 Output...")
				}

			} else if !begin && len(mergeMessage) > 200 {
				begin = true
				beginIndex = len(mergeMessage)
				logrus.Info("---------\n", "2 Output...")
			}

			if begin {
				if debug {
					logrus.Info(
						"-------------- H: S: ----------------\n[debug]: {H:"+strconv.Itoa(strings.LastIndex(mergeMessage, H))+"}, ",
						"{S:"+strconv.Itoa(strings.LastIndex(mergeMessage, S))+"}",
						"\n--------------------------------------")
				}
				// 遇到“H:”就结束接收
				if index := strings.LastIndex(mergeMessage, H); boH && index > -1 && index > beginIndex {
					logrus.Info("---------\n", cmdvars.I18n("H"))
					if idx := strings.LastIndex(self.Cache, H); idx >= 0 {
						self.Cache = self.Cache[:idx]
					}
					self.Closed = true
					return nil
				}
				// 遇到“System:”就结束接收
				if index := strings.LastIndex(mergeMessage, S); boS && index > -1 && index > beginIndex {
					logrus.Info("---------\n", cmdvars.I18n("S"))
					if idx := strings.LastIndex(self.Cache, S); idx >= 0 {
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

func handleClaudeError(err *clTypes.Claude2Error) (msg string) {
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

func responseClaudeError(ctx *gin.Context, err error, isStream bool, isCompletions bool, token string) {
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

	ResponseError(ctx, errMsg, isStream, isCompletions)
}
