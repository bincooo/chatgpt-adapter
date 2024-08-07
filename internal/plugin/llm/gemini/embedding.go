package gemini

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/pkg"
	"encoding/json"
	"io"
	"net/http"

	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func ConvertOpenAIRequestToGemini(openAIReq *pkg.EmbedRequest, model string) (*GeminiEmbedBatchReq, error) {
	if openAIReq.EncodingFormat != "" && openAIReq.EncodingFormat != "float" {
		return nil, errors.New("unsupported encoding format")
	}
	reqs := make([]GeminiEmbedReq, 0)
	switch v := openAIReq.Input.(type) {
	case string:
		reqs = append(reqs, GeminiEmbedReq{
			Model: model,
			Content: GeminiContent{
				Parts: []GeminiContPart{{Text: v}},
			},
		})
	case []interface{}:
		for _, text := range v {
			if t, ok := text.(string); ok {
				reqs = append(reqs, GeminiEmbedReq{
					Model: model,
					Content: GeminiContent{
						Parts: []GeminiContPart{{Text: t}},
					},
				})
			} else {
				return nil, errors.Errorf("unsupported input type: %T", t)
			}
		}
	default:
		return nil, errors.Errorf("unsupported input type: %T", v)
	}

	return &GeminiEmbedBatchReq{Requests: reqs}, nil
}

func ConvertGeminiResponseToOpenAI(geminiResp *GeminiResp, model string) *EmbedResponse {
	openAIResp := &EmbedResponse{
		Object: "list",
		Model:  model,
	}

	for i, geminiResp := range geminiResp.Embeddings {
		openAIResp.Data = append(openAIResp.Data, &EmbedResponseData{
			Object:    "embedding",
			Embedding: geminiResp.Values,
			Index:     i,
		})
	}

	openAIResp.Usage = &Usage{
		PromptTokens: 0,
		TotalTokens:  0,
	}

	return openAIResp
}

type GeminiEmbedBatchReq struct {
	Requests []GeminiEmbedReq `json:"requests"`
}

type GeminiEmbedReq struct {
	Model   string        `json:"model"`
	Content GeminiContent `json:"content"`
}

type GeminiContent struct {
	Parts []GeminiContPart `json:"parts"`
}

type GeminiContPart struct {
	Text string `json:"text"`
}

type EmbedResponseData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type EmbedResponse struct {
	Object string               `json:"object"`
	Data   []*EmbedResponseData `json:"data"`
	Model  string               `json:"model"`
	Usage  *Usage               `json:"usage"`
}

type GeminiResp struct {
	Embeddings []GeminiEmbedding `json:"embeddings"`
}

type GeminiEmbedding struct {
	Values []float32 `json:"values"`
}

func (API) Embedding(ctx *gin.Context) {

	openAIReq := common.GetGinEmbedding(ctx)
	var (
		token   = ctx.GetString("token")
		proxies = ctx.GetString("proxies")
	)

	geminiReq, err := ConvertOpenAIRequestToGemini(&openAIReq, openAIReq.Model)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}
	url := "https://generativelanguage.googleapis.com/v1beta/" +
		openAIReq.Model + ":batchEmbedContents?key=" + token
	resp, err := emit.ClientBuilder(plugin.HTTPClient).
		Proxies(proxies).
		Context(common.GetGinContext(ctx)).
		POST(url).
		JHeader().
		Body(geminiReq).DoC(emit.Status(http.StatusOK))

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var geminiResp GeminiResp
	json.Unmarshal(respBytes, &geminiResp)
	openAIResp := ConvertGeminiResponseToOpenAI(&geminiResp, openAIReq.Model)

	ctx.JSON(http.StatusOK, openAIResp)
}
