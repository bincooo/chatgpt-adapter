package response

import (
	"bufio"
	"bytes"
	"chatgpt-adapter/core/goja"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/logger"
	"chatgpt-adapter/core/tokenizer"
	"github.com/bincooo/coze-api"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/errors"
	"github.com/iocgo/sdk/stream"

	regexp "github.com/dlclark/regexp2"
	_ "github.com/iocgo/sdk"
)

var (
	CLAUDE_ROLE_FMT = func(role string) string { return fmt.Sprintf("\n\r\n%s: ", role) }
	GPT_ROLE_FMT    = func(role string) string { return fmt.Sprintf("<|start|>%s\n", role) }
	ROLE_FMT        = func(role string) string { return fmt.Sprintf("<|%s|>\n", role) }
	END             = "<|end|>\n\n"
	delimiter       = "\n\n"
)

var (
	regExp       = regexp.MustCompile(`^/(.+)/([a-z]*)$`, regexp.ECMAScript)
	regExpClears = []*regexp.Regexp{
		regexp.MustCompile(`<notes>\n*</notes>`, regexp.ECMAScript),
		regexp.MustCompile(`<example>\n*</example>`, regexp.ECMAScript),
		regexp.MustCompile(`\n{3,}`, regexp.ECMAScript),
	}
)

type ContentHolder struct{ env *env.Environment }

func (holder ContentHolder) Handle(ctx *gin.Context, completion model.Completion) (messages []model.Keyv[interface{}], err error) {
	schemas := []interface{}{
		"debug",       // 调试标记
		"toolChoice",  // 工具选择
		"echo",        // 不与AI交互，仅获取处理后的上下文
		"specialized", // 用于开启/关闭特化处理
	}
	content := strings.Join(stream.Map(stream.OfSlice(completion.Messages), join(false)).ToSlice(), delimiter)
	context := errors.New(func(e error) bool { err = e; return true })
	{
		content = errors.Try1(context, func() (str string, err error) {
			str, _, err = parseMessages[any](ctx, tokenizer.New(schemas...), content, nil)
			return
		})
	}

	// stream.OfSlice(handlers).Range(func(handler regexHandler) { content = handler(content) })
	specialized := ctx.GetBool("specialized") // 是否开启特化处理
	if specialized {
		if IsClaude(ctx, ctx.GetString("token"), completion.Model) {
			messages, err = goja.ParseMessages(splitToMessages(content, false), "txt")
			return
		}
	}

	messages = splitToMessages(content, true)
	return
}

func parseMessages[T any](ctx *gin.Context, parser *tokenizer.Parser, content string, exec func(elem tokenizer.Elem, clean func()) T) (result string, handlers []T, err error) {
	result = content
	elems := parser.Parse(result)
	clean := func(index int) {
		if index < 0 || index >= len(elems) {
			return
		}
		elems = append(elems[:index], elems[index+1:]...)
	}

	if exec == nil {
		exec = func(tokenizer.Elem, func()) (zero T) { return }
	}

	specialized := env.Env.GetBool("specialized")
	ctx.Set("specialized", specialized)

	for i := len(elems) - 1; i > 0; i-- {
		elem := elems[i]
		if elem.Kind() != tokenizer.Ident {
			continue
		}

		// debug 模式
		if elem.Label() == "debug" {
			ctx.Set(vars.GinDebugger, true)
			clean(i)
			continue
		}

		// 不与AI交互，仅获取处理后的上下文
		if elem.Label() == "echo" {
			ctx.Set(vars.GinEcho, true)
			clean(i)
			continue
		}

		if elem.Label() == "toolChoice" {
			id := "-1"
			tasks := false
			enabled := false
			if value, ok := elem.Str("id"); ok {
				id = value
			}
			if value, ok := elem.Boolean("tasks"); ok {
				tasks = value
			}
			if value, ok := elem.Boolean("enabled"); ok {
				enabled = value
			}

			clean(i)
			ctx.Set(vars.GinTool, model.Keyv[interface{}]{
				"id":      id,
				"tasks":   tasks,
				"enabled": enabled,
			})
			continue
		}

		// 特化处理
		if elem.Label() == "specialized" {
			value, ok := elem.Boolean("enabled")
			if ok {
				specialized = value
			}
			clean(i)
			ctx.Set("specialized", specialized)
			continue
		}

		t := exec(elem, func() { clean(i) })
		if !common.IsNIL(t) {
			handlers = append(handlers, t)
		}
	}

	slices.Reverse(handlers)
	result = tokenizer.JoinString(elems)
	return
}

func ConvertRole(ctx *gin.Context, role string) (newRole, end string) {
	completion := common.GetGinCompletion(ctx)
	if IsClaude(ctx, "", completion.Model) {
		switch role {
		case "user":
			newRole = CLAUDE_ROLE_FMT("Human")
		case "assistant":
			newRole = CLAUDE_ROLE_FMT("Assistant")
		default:
			newRole = CLAUDE_ROLE_FMT("SYSTEM")
		}
		return
	}

	end = END
	if IsGPT(completion.Model) {
		switch role {
		case "user", "assistant":
			newRole = GPT_ROLE_FMT(role)
		default:
			newRole = GPT_ROLE_FMT("system")
		}
		return
	}

	newRole = ROLE_FMT(role)
	return
}

func splitToMessages(content string, merge bool) (messages []model.Keyv[interface{}]) {
	chunkMap := map[string][]byte{
		"assistant": []byte("\n\nassistant: "),
		"user":      []byte("\n\nuser: "),
		"system":    []byte("\n\nsystem: "),
	}

	scanner := bufio.NewScanner(bytes.NewBuffer([]byte(content)))
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return
		}
		pos := make([]int, 0)
		for _, chunk := range chunkMap {
			if i := bytes.Index(data, chunk); i >= 0 {
				pos = append(pos, i)
			}
		}

		if len(pos) > 0 {
			slices.Sort(pos)
			return pos[0] + 2, data[0:pos[0]], nil
		}

		if atEOF {
			return len(data), data, nil
		}
		return
	})

	Add := func(message model.Keyv[interface{}]) {
		if message == nil {
			return
		}
		if message.IsE("content") {
			return
		}
		messages = append(messages, message)
	}

	jo := func(v1, v2 string) string {
		for _, ex := range regExpClears {
			v2, _ = ex.Replace(v2, "", 0, -1)
		}
		v2 = strings.TrimSpace(v2)
		if v1 == "" {
			return v2
		}
		return v1 + delimiter + v2
	}

	message := make(model.Keyv[interface{}])
	for scanner.Scan() {
		chunkBytes := scanner.Bytes()
		if len(chunkBytes) == 0 {
			continue
		}

		role := ""
		for r, chunk := range chunkMap {
			if bytes.HasPrefix(chunkBytes, chunk[2:]) {
				chunkBytes = chunkBytes[len(chunk[2:]):]
				role = r
				break
			}
		}

		if role == "" || message.IsE("role") || message.Is("role", role) {
			if message.IsE("role") {
				role = elseOf(role != "", role, "user")
				message.Set("role", role)
			}

			if merge {
				message.Set("content", jo(message.GetString("content"), string(chunkBytes)))
				continue
			}
		}

		Add(message)
		message = make(model.Keyv[interface{}])
		message.Set("role", role)
		message.Set("content", jo("", string(chunkBytes)))
	}

	Add(message)
	return
}

func IsGPT(model string) bool {
	model = strings.ToLower(model)
	return strings.Contains(model, "openai") || strings.Contains(model, "gpt")
}

func IsClaude(ctx *gin.Context, token, model string) bool {
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
		if len(values) > 3 && "w" == values[3] && strings.Contains(token, "[claude=true]") {
			ctx.Set(key, true)
			return true
		}

		return false
	}

	return isc
}

func At(str string) (ok bool) {
	if len(str) < 1 {
		return
	}
	if str[0] != '@' {
		return
	}
	_, err := strconv.Atoi(str[1:])
	return err == nil
}

func regexScope(regex string) (re string) {
	scope := ""
	matched, err := regExp.FindStringMatch(strings.TrimSpace(regex))
	if err != nil {
		logger.Warn(err)
		return regex
	}
	if matched == nil {
		return regex
	}
	regex = matched.GroupByNumber(1).String()
	scope = matched.GroupByNumber(2).String()

	if strings.Contains(scope, "s") {
		re += "s"
	}
	if strings.Contains(scope, "m") {
		re += "m"
	}
	if strings.Contains(scope, "i") {
		re += "i"
	}
	if len(re) > 0 {
		re = "(?" + re + ")"
	}
	re += regex
	return
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}

func join(clear bool) func(model.Keyv[interface{}]) string {
	return func(keyv model.Keyv[interface{}]) string {
		content := strings.TrimSpace(keyv.GetString("content"))
		if content == "" {
			return ""
		}

		if clear {
			for _, ex := range regExpClears {
				content, _ = ex.Replace(content, delimiter, 0, -1)
			}
		}

		return fmt.Sprintf("%s: %s", keyv.GetString("role"), content)
	}
}

func joinT(keyv model.Keyv[interface{}]) string {
	content := strings.TrimSpace(keyv.GetString("content"))
	return fmt.Sprintf("%s: %s", keyv.GetString("role"), content)
}
