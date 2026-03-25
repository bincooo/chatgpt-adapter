package emulate

import (
	"github.com/google/uuid"
)

type RequestStructure struct {
	Action           string `json:"action"`
	ConversationMode struct {
		Kind    string `json:"kind"`
		GizmoId string `json:"gizmo_id,omitempty"`
	} `json:"conversation_mode"`
	Messages                   []Message `json:"messages,omitempty"`
	ParentMessageID            string    `json:"parent_message_id,omitempty"`
	ConversationID             string    `json:"conversation_id,omitempty"`
	Model                      string    `json:"model"`
	HistoryAndTrainingDisabled bool      `json:"history_and_training_disabled"`
	WebsocketRequestId         string    `json:"websocket_request_id"`
	ForceSSE                   bool      `json:"force_use_sse"`

	Token      string `json:"-"`
	ProofToken string `json:"-"`
}

type Message struct {
	ID     uuid.UUID `json:"id"`
	Author struct {
		Role string `json:"role"`
	} `json:"author"`
	Content struct {
		ContentType string        `json:"content_type"`
		Parts       []interface{} `json:"parts"`
	} `json:"content"`
	Metadata *struct {
		Attachments []AttachmentMeta `json:"attachments,omitempty"`
	} `json:"metadata,omitempty"`
}

type AttachmentMeta struct {
	Id        string `json:"id"`
	MimeType  string `json:"mimeType"`
	Name      string `json:"name"`
	Size      int    `json:"size"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	TokenSize int    `json:"file_token_size,omitempty"`
}

type multimodel struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Image struct {
		Url string `json:"url"`
	} `json:"image_url,omitempty"`
}

func newChatGPTRequest() *RequestStructure {
	return &RequestStructure{
		Action:                     "next",
		ParentMessageID:            uuid.NewString(),
		Model:                      "auto",
		HistoryAndTrainingDisabled: true,
		ConversationMode: struct {
			Kind    string `json:"kind"`
			GizmoId string `json:"gizmo_id,omitempty"`
		}{
			Kind: "primary_assistant",
		},
		WebsocketRequestId: uuid.NewString(),
		ForceSSE:           true,
	}
}

func (req *RequestStructure) addMessage(role string, content interface{}) {
	var parts []interface{}
	msgType := "text"
	switch v := content.(type) {
	case string:
		parts = append(parts, v)
	case []interface{}:
		var items []multimodel
		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			typed, _ := itemMap["type"].(string)
			if typed == "text" {
				text, _ := itemMap["text"].(string)
				items = append(items, multimodel{Type: typed, Text: text})
			}
		}
		for _, item := range items {
			parts = append(parts, item.Text)
		}
	}

	var msg = Message{
		ID: uuid.New(),
		Author: struct {
			Role string `json:"role"`
		}{Role: role},
		Content: struct {
			ContentType string        `json:"content_type"`
			Parts       []interface{} `json:"parts"`
		}{ContentType: msgType, Parts: parts},
		Metadata: nil,
	}

	req.Messages = append(req.Messages, msg)
}
