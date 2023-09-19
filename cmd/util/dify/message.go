package dify

import (
	cmdtypes "github.com/bincooo/AutoAI/cmd/types"
	"regexp"
	"strings"
)

func ConvertMessages(r *cmdtypes.RequestDTO) {
	handle := func(val string) map[string]string {
		val = strings.TrimSpace(val)
		if strings.HasPrefix(val, "Assistant:") {
			return map[string]string{
				"role":    "assistant",
				"content": strings.TrimSpace(strings.TrimPrefix(val, "Assistant:")),
			}
		}
		if strings.HasPrefix(val, "System:") {
			return map[string]string{
				"role":    "system",
				"content": strings.TrimSpace(strings.TrimPrefix(val, "System:")),
			}
		}
		return map[string]string{
			"role":    "user",
			"content": strings.TrimSpace(strings.TrimPrefix(val, "Human:")),
		}
	}

	var subContext string
	content := r.Messages[0]["content"]
	subContext, content = CutSubContext(content)
	content = strings.ReplaceAll(content, "<histories></histories>", "")
	content = strings.TrimSuffix(content, "\nAssistant: ")

	content = strings.ReplaceAll(content, "<histories>", "<|[1]|><histories>")
	contents := strings.Split(content, "<|[1]|>")
	temp := contents
	contents = []string{}
	for _, human := range temp {
		if human == "" {
			continue
		}
		histories := strings.Split(human, "</histories>")
		contents = append(contents, histories...)
	}

	splitHandle := func(item string) []map[string]string {
		messages := make([]map[string]string, 0)
		item = strings.ReplaceAll(item, "\nHuman:", "<|[1]|>\nHuman:")
		humans := strings.Split(item, "<|[1]|>\n")
		temp = humans
		humans = []string{}
		for _, human := range temp {
			if human == "" {
				continue
			}
			human = strings.ReplaceAll(human, "\nAssistant:", "<|[1]|>\nAssistant:")
			assistants := strings.Split(human, "<|[1]|>\n")
			humans = append(humans, assistants...)
		}
		temp = humans
		humans = []string{}
		for _, human := range temp {
			if human == "" {
				continue
			}
			human = strings.ReplaceAll(human, "\nSystem:", "<|[1]|>\nSystem:")
			systems := strings.Split(human, "<|[1]|>\n")
			humans = append(humans, systems...)
		}
		for _, human := range humans {
			if human == "" {
				continue
			}
			messages = append(messages, handle(human))
		}
		return messages
	}

	messages := make([]map[string]string, 0)
	for _, item := range contents {
		if item == "Assistant: " || item == "Here is the chat histories between human and assistant, inside <histories></histories> XML tags." {
			continue
		}
		if strings.HasPrefix(item, "<histories>") {
			item = strings.TrimPrefix(item, "<histories>")
			item = strings.TrimSuffix(item, "</histories>")
			messages = append(messages, splitHandle(item)...)
			continue
		}
		messages = append(messages, splitHandle(item)...)
	}
	if l := len(messages); subContext != "" && l > 0 {
		messages[0]["content"] += "\n\n" + subContext
	}
	r.Messages = messages
}

func CutSubContext(content string) (subContext string, result string) {
	compileRegex := regexp.MustCompile("^[a-zA-Z]+[^.]+XML tags\\.")
	matchArr := compileRegex.FindStringSubmatch(content)
	if len(matchArr) == 0 {
		return "", content
	}

	xmltags := matchArr[0]
	content = strings.Replace(content, xmltags, "", -1)

	idx := strings.Index(content, "<context>")
	rIdx := strings.LastIndex(content, "</context>")
	if idx > 0 && rIdx > idx {
		c := content[idx : rIdx+10]
		content = strings.Replace(content, c, "", -1)
		sIdx := strings.Index(content, "System:")
		hIdx := strings.Index(content, "<histories>")
		if sIdx > 0 && (hIdx == -1 || sIdx < hIdx) {
			content = strings.Replace(content, "System:", "", sIdx)
			content = "System:" + content
		}
		return xmltags + "\n" + c, strings.TrimSpace(content)
	} else {
		return "", content
	}
}
