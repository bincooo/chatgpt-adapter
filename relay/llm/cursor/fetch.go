package cursor

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/iocgo/sdk/stream"
	"math/rand"
	"net/http"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"github.com/bincooo/emit.io"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	. "chatgpt-adapter/relay/llm/cursor/proto"
)

func fetch(ctx context.Context, proxied string, cookie string, buffer []byte) (response *http.Response, err error) {
	checksum := fmt.Sprintf("zo%s%s/%s", getRandId(6, "max"), getRandId(64, "max"), getRandId(64, "max"))
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST("https://api2.cursor.sh/aiserver.v1.AiService/StreamChat").
		Header("authorization", "Bearer "+cookie).
		Header("content-type", "application/connect+proto").
		Header("connect-accept-encoding", "gzip,br").
		Header("connect-protocol-version", "1").
		Header("user-agent", "connect-es/1.4.0").
		Header("x-amzn-trace-id", "Root="+uuid.NewString()).
		Header("x-cursor-checksum", checksum).
		Header("x-cursor-client-version", "0.42.3").
		Header("x-cursor-timezone", "Asia/Shanghai").
		Header("x-ghost-mode", "false").
		Header("x-request-id", uuid.New().String()).
		Header("host", "api2.cursor.sh").
		Bytes(buffer).
		DoC(emit.Status(http.StatusOK), emit.IsPROTO)
	return
}

func convertRequest(completion model.Completion) (buffer []byte, err error) {
	messages := stream.Map(stream.OfSlice(completion.Messages), func(message model.Keyv[interface{}]) *ChatMessage_UserMessage {
		return &ChatMessage_UserMessage{
			MessageId: uuid.NewString(),
			Role:      elseOf[int32](message.Is("role", "user"), 1, 2),
			Content:   message.GetString("content"),
		}
	}).ToSlice()
	message := &ChatMessage{
		Messages: messages,
		Instructions: &ChatMessage_Instructions{
			Instruction: "",
		},
		ProjectPath: "/path/to/project",
		Model: &ChatMessage_Model{
			Name:  completion.Model[7:],
			Empty: "",
		},
		Summary:        "",
		RequestId:      uuid.NewString(),
		ConversationId: uuid.NewString(),
	}

	protoBytes, err := proto.Marshal(message)
	if err != nil {
		return
	}

	header := int32ToBytes(0, uint32(len(protoBytes)))
	buffer = append(header, protoBytes...)
	return
}

func getRandId(size int, dictType string) string {
	customDict := ""
	switch dictType {
	case "alphabet":
		customDict = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	case "max":
		customDict = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_-"
	default:
		customDict = "0123456789"
	}

	buf := make([]byte, 0)
	for range size {
		buf = append(buf, customDict[rand.Intn(len(customDict))])
	}
	return string(buf)
}

func int32ToBytes(magic byte, num uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, num)
	return append([]byte{magic}, bytes...)
}

func elseOf[T any](condition bool, a1, a2 T) T {
	if condition {
		return a1
	}
	return a2
}
