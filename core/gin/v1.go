package gin

import (
	"chatgpt-adapter/core/common/toolcall"
	"chatgpt-adapter/core/common/vars"
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk"
	"time"
)

const ginTokens = "__tokens__"

// @Router()
type Handler struct{ extensions []inter.Adapter }

// @Inject()
func New(container *sdk.Container) *Handler {
	extensions := sdk.ListInvokeAs[inter.Adapter](container)
	return &Handler{extensions}
}

// @GET(path = "/")
func (h *Handler) index(gtx *gin.Context) {
	gtx.Writer.WriteString("<div style='color:green'>success ~</div>")
}

// @POST(path = "
//
//	v1/chat/completions,
//	v1/object/completions,
//	proxies/v1/chat/completions
//
// ")
func (h *Handler) completions(gtx *gin.Context) {
	var completion model.Completion
	if err := gtx.BindJSON(&completion); err != nil {
		logger.Error(err)
		response.Error(gtx, -1, err)
		return
	}

	gtx.Set(vars.GinCompletion, completion)
	logger.Infof("curr model: %s", completion.Model)
	if !response.MessageValidator(gtx) {
		return
	}

	for _, extension := range h.extensions {
		ok, err := extension.Match(gtx, completion.Model)
		if err != nil {
			response.Error(gtx, -1, err)
			return
		}
		if !ok {
			continue
		}

		gtx.Set(vars.GinMatchers, response.NewMatchers(gtx, func(t byte, str string) {
			if completion.Stream && t == 0 {
				response.SSEResponse(gtx, "matcher", str, time.Now().Unix())
			}
			if completion.Stream && t == 1 {
				response.ReasonSSEResponse(gtx, "matcher", "", str, time.Now().Unix())
			}
		}))

		messages, err := extension.HandleMessages(gtx, completion)
		if err != nil {
			logger.Error("Error handling messages: ", err)
			response.Error(gtx, 500, err)
			return
		}

		if gtx.GetInt(ginTokens) == 0 {
			calcTokens(gtx, messages)
		}

		completion.Messages = messages
		gtx.Set(vars.GinCompletion, completion)

		if toolcall.NeedExec(gtx) {
			if ok, err = extension.ToolChoice(gtx); err != nil {
				response.Error(gtx, -1, err)
				return
			}
			if ok {
				return
			}
		}

		if err = extension.Completion(gtx); err != nil {
			response.Error(gtx, -1, err)
		}
		return
	}
	response.Error(gtx, -1, fmt.Sprintf("model '%s' is not not yet supported", completion.Model))
}

func calcTokens(gtx *gin.Context, messages []model.Keyv[interface{}]) {
	tokens := 0
	for _, message := range messages {
		if !message.IsString("content") {
			continue
		}
		value := message.GetString("content")
		tokens += response.CalcTokens(value)
	}
	gtx.Set(ginTokens, tokens)
}

// @POST(path = "
//
//	/v1/embeddings,
//	proxies/v1/embeddings
//
// ")
func (h *Handler) embeddings(gtx *gin.Context) {
	var embed model.Embed
	if err := gtx.BindJSON(&embed); err != nil {
		logger.Error(err)
		response.Error(gtx, -1, err)
		return
	}

	gtx.Set(vars.GinEmbedding, embed)
	logger.Infof("curr model: %s", embed.Model)
	for _, extension := range h.extensions {
		ok, err := extension.Match(gtx, embed.Model)
		if err != nil {
			response.Error(gtx, -1, err)
			return
		}
		if ok {
			if err = extension.Embedding(gtx); err != nil {
				response.Error(gtx, -1, err)
			}
			return
		}
	}
	response.Error(gtx, -1, fmt.Sprintf("model '%s' is not not yet supported", embed.Model))
}

// @POST(path = "
//
//	v1/images/generations,
//	v1/object/generations,
//	proxies/v1/images/generations
//
// ")
func (h *Handler) generations(gtx *gin.Context) {
	var generation model.Generation
	if err := gtx.BindJSON(&generation); err != nil {
		response.Error(gtx, 500, err)
		return
	}

	gtx.Set(vars.GinGeneration, generation)
	for _, extension := range h.extensions {
		ok, err := extension.Match(gtx, generation.Model)
		if err != nil {
			response.Error(gtx, 500, err)
			return
		}
		if ok {
			if err = extension.Generation(gtx); err != nil {
				response.Error(gtx, -1, err)
			}
			return
		}
	}
	response.Error(gtx, -1, fmt.Sprintf("model '%s' is not not yet supported", generation.Model))
}

// @GET(path = "
//
//	v1/models,
//	proxies/v1/models
//
// ")
func (h *Handler) models(gtx *gin.Context) {
	models := make([]model.Model, 0)
	for _, extension := range h.extensions {
		models = append(models, extension.Models()...)
	}
	gtx.JSON(200, gin.H{
		"object": "list",
		"data":   models,
	})
}
