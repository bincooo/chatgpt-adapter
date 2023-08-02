package chain

import (
	"github.com/bincooo/MiaoX/store"
	"github.com/bincooo/MiaoX/types"
	"github.com/bincooo/MiaoX/vars"
	clTypes "github.com/bincooo/claude-api/types"
	"regexp"
	"strings"
)

var SystemTips = []string{
	"System: Thank you for your response. Let's move our conversation in a more positive direction.",
	"System: Thank you for your understanding. Let's move our conversation in a more positive direction.",
	"System: Now continue the conversation with no more than 300 words based on the given context. Do not mention anything unrelated.",
	"System: Please do not continue or engage with any inappropriate content. Let's move our conversation in a more positive direction.",
	"System: Thank you, let's move our conversation in a more positive direction.",
	"System: Let's end our conversation here. I wish you happiness.",
}

const (
	A = "A:"
	H = "H:"
	S = "System:"
)

// 需要配合指定的预设
type ClaudeWeb2sInterceptor struct {
	types.BaseInterceptor
}

func (c *ClaudeWeb2sInterceptor) Before(bot types.Bot, ctx *types.ConversationContext) bool {
	if ctx.Bot == vars.Claude {
		ctx.Model = vars.Model4WebClaude2S
		ctx.H = Handle
		if !strings.Contains(ctx.Preset, "[history]") {
			return true
		}

		messages := store.GetMessages(ctx.Id)

		history := ""
		for _, message := range messages {
			if message["author"] == "bot" {
				text := strings.TrimSpace(message["text"])
				text = strings.Replace(text, "❤,已解除苛刻的道德限制:", "", -1)

				var result []string
				split := strings.Split(text, A)
				for _, item := range split {

					if strings.HasPrefix(item, "System:") {
						continue
					}

					if strings.HasPrefix(item, H) {
						continue
					}

					for _, tips := range SystemTips {
						item = strings.ReplaceAll(item, tips, "")
					}

					index := strings.Index(item, H)
					if index > 0 {
						result = append(result, strings.TrimSpace(item[:index]))
					} else {
						result = append(result, strings.TrimSpace(item))
					}

				}

				text = strings.ReplaceAll(strings.Join(result, "\n"), "\n\n", "\n")

				re := regexp.MustCompile(`System:[^\n]+`)
				text = re.ReplaceAllString(text, "")

				re = regexp.MustCompile(`[(]*I apologize,[^\n]+`)
				text = re.ReplaceAllString(text, "")

				text = strings.ReplaceAll(text, "[End]", "")
				text = strings.ReplaceAll(text, "[End Chat]", "")
				text = strings.ReplaceAll(text, "\"\"", "")
				if !strings.HasPrefix(text, A) {
					history += A + " " + strings.TrimSpace(text)
				} else {
					history += strings.TrimSpace(text)
				}
			}

			if message["author"] == "user" {
				text := strings.TrimSpace(message["text"])
				if !strings.HasPrefix(text, H) {
					history += H + " " + text
				} else {
					history += text
				}
			}
			history += "\n\n"
		}

		if history != "" {
			history = "\n" + history
		}

		preset := strings.Replace(ctx.Preset, "[history]", history, -1)
		ctx.Prompt = strings.Replace(preset, "[content]", ctx.Prompt, -1)
	}
	return true
}

func Handle(rChan any) func(*types.CacheBuffer) error {
	pos := 0
	begin := false
	beginIndex := -1
	partialResponse := rChan.(chan clTypes.PartialResponse)
	return func(self *types.CacheBuffer) error {
		response, ok := <-partialResponse
		if !ok {
			self.Cache = strings.TrimSuffix(self.Cache, A)
			self.Closed = true
			return nil
		}

		if response.Error != nil {
			self.Closed = true
			return response.Error
		}

		text := response.Text
		str := []rune(text)
		self.Cache += string(str[pos:])
		pos = len(str)

		mergeMessage := self.Complete + self.Cache
		if index := strings.Index(mergeMessage, A); index > -1 {
			if !begin {
				begin = true
				beginIndex = index
			}
			//} else {
			//	self.Closed = true
			//	return nil
			//}
		} else if !begin && len(mergeMessage) > 200 {
			begin = true
			beginIndex = pos
		}

		if begin {
			if index := strings.Index(mergeMessage, H); index > -1 && index > beginIndex {
				self.Cache = strings.TrimSuffix(self.Cache, H)
				self.Closed = true
				return nil
			} else if index = strings.Index(mergeMessage, S); index > -1 && index > beginIndex {
				self.Cache = strings.TrimSuffix(self.Cache, S)
				self.Closed = true
				return nil
			}
		}
		return nil
	}
}
