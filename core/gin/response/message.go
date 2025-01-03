package response

import (
	"fmt"
	"strings"

	"chatgpt-adapter/core/common"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	_ "github.com/iocgo/sdk"
	"github.com/iocgo/sdk/env"
)

const (
	END = "<|end|>\n\n"
)

func defaultRole(role string) string { return fmt.Sprintf("<|%s|>\n", role) }
func claudeRole(role string) string  { return fmt.Sprintf("\n\r\n%s: ", role) }
func gptRole(role string) string     { return fmt.Sprintf("<|start|>%s\n", role) }
func bingRole(role string) string {
	switch role {
	case "user":
		return "Human: "
	case "assistant":
		return "Ai: "
	default:
		return "Instructions: "
	}
}

func ConvertRole(ctx *gin.Context, role string) (newRole, end string) {
	completion := common.GetGinCompletion(ctx)
	if IsClaude(ctx, completion.Model) {
		switch role {
		case "user":
			newRole = claudeRole("Human")
		case "assistant":
			newRole = claudeRole("Assistant")
		default:
			newRole = claudeRole("SYSTEM")
		}
		return
	}

	if IsBing(completion.Model) {
		newRole = bingRole(role)
		return
	}

	end = END
	if IsGPT(completion.Model) {
		switch role {
		case "user", "assistant":
			newRole = gptRole(role)
		default:
			newRole = gptRole("system")
		}
		return
	}

	newRole = defaultRole(role)
	return
}

func IsBing(mod string) bool {
	return mod == "bing"
}

func IsGPT(model string) bool {
	model = strings.ToLower(model)
	return strings.Contains(model, "openai") || strings.Contains(model, "gpt")
}

func IsClaude(ctx *gin.Context, model string) bool {
	key := "__is-claude__"
	if ctx.GetBool(key) {
		return true
	}

	if model == "coze/websdk" || common.IsGinCozeWebsdk(ctx) {
		model = env.Env.GetString("coze.websdk.model")
		return model == coze.ModelClaude35Sonnet_200k || model == coze.ModelClaude3Haiku_200k
	}

	isc := strings.Contains(strings.ToLower(model), "claude")
	if isc {
		ctx.Set(key, true)
		return true
	}

	if strings.HasPrefix(model, "coze/") {
		values := strings.Split(model[5:], "-")
		if len(values) > 3 && "w" == values[3] && strings.Contains(ctx.GetString("token"), "[claude=true]") {
			ctx.Set(key, true)
			return true
		}

		return false
	}

	return isc
}