package cursor

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net/http"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/stream"
)

var (
	g_checksum = ""
)

func fetch(ctx *gin.Context, env *env.Environment, cookie string, buffer []byte) (response *http.Response, err error) {
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(env.GetString("server.proxied")).
		POST("https://api2.cursor.sh/aiserver.v1.AiService/StreamChat").
		Header("authorization", "Bearer "+cookie).
		Header("content-type", "application/connect+proto").
		Header("connect-accept-encoding", "gzip,br").
		Header("connect-protocol-version", "1").
		Header("user-agent", "connect-es/1.4.0").
		Header("x-cursor-checksum", genChecksum(ctx, env)).
		Header("x-cursor-client-version", "0.42.3").
		Header("x-cursor-timezone", "Asia/Shanghai").
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

	header := int32ToBytes(0, len(protoBytes))
	buffer = append(header, protoBytes...)
	return
}

func genChecksum(ctx *gin.Context, env *env.Environment) string {
	checksum := ctx.GetHeader("x-cursor-checksum")
	if checksum == "" {
		checksum = env.GetString("cursor.checksum")
	}
	if checksum == "" {
		if g_checksum == "" {
			g_checksum = fmt.Sprintf("zo%s%s/%s", randId(6, "max"), randId(64, "max"), randId(64, "max"))
		}
		checksum = g_checksum
	}
	return checksum
}

func randId(size int, dictType string) string {
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

func int32ToBytes(magic byte, num int) []byte {
	hex := make([]byte, 4)
	binary.BigEndian.PutUint32(hex, uint32(num))
	return append([]byte{magic}, hex...)
}

func bytesToInt32(hex []byte) int {
	return int(binary.BigEndian.Uint32(hex))
}

func elseOf[T any](condition bool, a1, a2 T) T {
	if condition {
		return a1
	}
	return a2
}
