package chatgpt

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"bypass/llm/chatgpt/emulate"

	"github.com/xllm-go/g/model"
)

func fetch(ctx *model.Ctx) (reader io.Reader, err error) {
	completion := ctx.GetCompletion()

	emulator := emulate.GetEmulator()
	message, err := emulator.ConvertAPIRequest(completion)
	if err != nil {
		return
	}

	chunk, err := json.Marshal(message)
	if err != nil {
		return
	}

	request, err := http.NewRequest(http.MethodPost, "https://chatgpt.com/backend-anon/conversation", bytes.NewReader(chunk))
	if err != nil {
		return
	}

	request.Header.Set("User-Agent", emulator.Equi())
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Oai-Device-Id", emulator.GetDeviceId())
	request.Header.Set("Oai-Language", "en-US")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://chatgpt.com")
	request.Header.Set("Referer", "https://chatgpt.com/")
	if message.Token != "" {
		request.Header.Set("Openai-Sentinel-Chat-Requirements-Token", message.Token)
	}
	if message.ProofToken != "" {
		request.Header.Set("Openai-Sentinel-Proof-Token", message.ProofToken)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	reader = response.Body
	return
}
