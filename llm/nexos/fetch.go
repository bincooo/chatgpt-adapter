package nexos

import (
	"bypass/jinja"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xllm-go/g/logger"
	"github.com/xllm-go/g/model"
)

const (
	baseURL   = "https://workspace.nexos.ai"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0"
)

type requestBody struct {
	ComparisonId     string `json:"comparisonId"`
	ComparisonItemId string `json:"comparisonItemId"`
	UserPrompt       struct {
		Role               string   `json:"role"`
		Content            string   `json:"content"`
		MessageUuid        string   `json:"messageUuid"`
		OutputCapabilities []string `json:"outputCapabilities"`
		Timestamp          int64    `json:"timestamp"`
		ModelId            string   `json:"modelId"`
	} `json:"userPrompt"`
}

func fetch(ctx *model.Ctx) (reader io.Reader, err error) {
	completion := ctx.GetCompletion()
	message, err := model.JinjaMessage(jinja.DefaultTemplate, completion)
	if err != nil {
		return
	}

	modelId := modelMap[completion.Model[6:]]
	comparisonId, comparisonItemId, err := createChatId(ctx.Token, modelId)
	if err != nil {
		return
	}

	chunk, _ := json.Marshal(requestBody{
		ComparisonId:     comparisonId,
		ComparisonItemId: comparisonItemId,
		UserPrompt: struct {
			Role               string   `json:"role"`
			Content            string   `json:"content"`
			MessageUuid        string   `json:"messageUuid"`
			OutputCapabilities []string `json:"outputCapabilities"`
			Timestamp          int64    `json:"timestamp"`
			ModelId            string   `json:"modelId"`
		}{
			Role:               "user",
			Content:            message,
			MessageUuid:        uuid.NewString(),
			OutputCapabilities: []string{"text"},
			Timestamp:          time.Now().UnixNano(),
			ModelId:            modelId,
		},
	})

	request, err := http.NewRequest("POST", "https://workspace.nexos.ai/api/stream-comparison-completions", bytes.NewBuffer(chunk))
	if err != nil {
		return
	}

	setHeaders(request, ctx.Token)
	request.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	request.Header.Set("Referer", "https://workspace.nexos.ai/"+comparisons+"/comparisons/"+comparisonId)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	if response.StatusCode != 200 {
		chunk, _ = io.ReadAll(response.Body)
		err = errors.New(string(chunk))
		_ = response.Body.Close()
		return
	}

	reader = response.Body
	return
}

func createChatId(cookie, model string) (comparisonId, comparisonItemId string, err error) {
	redirect, err := createChatRedirect(cookie, model)
	if err != nil {
		return
	}

	request, err := http.NewRequest("GET", "https://workspace.nexos.ai"+redirect+".data", nil)
	if err != nil {
		logger.Sugar().Error(err)
		return
	}

	setHeaders(request, cookie)
	request.Header.Set("Referer", "https://workspace.nexos.ai/"+comparisons+"/comparisons")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if response.StatusCode != 200 {
		err = errors.New(response.Status)
		return
	}

	chunk, _ := io.ReadAll(response.Body)
	var arr []interface{}
	err = json.Unmarshal(chunk, &arr)
	if err != nil {
		return
	}

	for idx := range len(arr) - 1 {
		value := arr[idx]
		str, ok := value.(string)
		if ok && str == "firstUserMessageContent" {
			comparisonId = strings.Split(redirect, "/comparisons/")[1]
			comparisonItemId = arr[len(arr)-1].(string)
			return
		}
	}

	err = errors.New("chatId not found")
	return
}

func createChatRedirect(cookie, model string) (redirect string, err error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/comparisons/new.data?ids=cca533ac-85fa-420d-bb56-692206442322,%s", baseURL, comparisons, model), nil)
	if err != nil {
		return
	}

	setHeaders(request, cookie)
	request.Header.Set("Referer", "https://workspace.nexos.ai/"+comparisons+"/comparisons")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if response.StatusCode != 202 {
		err = errors.New(response.Status)
		return
	}

	chunk, err := io.ReadAll(response.Body)
	var arr []interface{}
	err = json.Unmarshal(chunk, &arr)
	if err != nil {
		return
	}

	for idx := range len(arr) - 1 {
		value := arr[idx]
		str, ok := value.(string)
		if ok && str == "redirect" {
			redirect = arr[idx+1].(string)
			return
		}
	}

	err = errors.New("redirect not found")
	return
}

func setHeaders(request *http.Request, cookie string) {
	u, _ := url.Parse(baseURL)
	request.Header.Set("Cookie", cookie)
	request.Header.Set("Host", u.Host)
	request.Header.Set("Origin", baseURL)
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Accept", "*/*")
}
