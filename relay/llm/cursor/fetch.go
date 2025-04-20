package cursor

import (
	"chatgpt-adapter/core/cache"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/logger"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/stream"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func fetch(ctx *gin.Context, env *env.Environment, cookie string, buffer []byte) (response *http.Response, err error) {
	//count, err := checkUsage(ctx, env, 150)
	//if err != nil {
	//	return
	//}
	//if count <= 0 {
	//	err = fmt.Errorf("invalid usage")
	//	return
	//}
	key := uuid.NewString()
	message := &BidiAppend{
		Chunk: hex.EncodeToString(buffer),
		SessionKey: &BidiAppend_SessionKey{
			Value: key,
		},
	}

	buffer, err = proto.Marshal(message)
	if err != nil {
		return
	}

	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(env.GetString("server.proxied")).
		POST("https://api2.cursor.sh/aiserver.v1.BidiService/BidiAppend").
		Header("authorization", "Bearer "+cookie).
		Header("content-type", "application/connect+proto").
		Header("connect-accept-encoding", "gzip").
		Header("connect-content-encoding", "gzip").
		Header("connect-protocol-version", "1").
		Header("traceparent", "00-"+strings.ReplaceAll(uuid.NewString(), "-", "")+"-"+common.Hex(16)+"-00").
		Header("user-agent", "connect-es/1.6.1").
		Header("x-amzn-trace-id", "Root="+uuid.NewString()).
		Header("x-client-key", genClientKey(ctx.GetString("token"))).
		Header("x-cursor-checksum", genChecksum(ctx, env)).
		Header("x-cursor-client-version", "0.48.9").
		Header("x-cursor-timezone", "Asia/Shanghai").
		Header("x-ghost-mode", "false").
		Header("x-request-id", uuid.NewString()).
		Header("x-session-id", uuid.NewString()).
		Header("host", "api2.cursor.sh").
		Header("Connection", "close").
		Header("Transfer-Encoding", "chunked").
		Bytes(buffer).
		DoC(emit.Status(http.StatusOK), emit.IsPROTO)
	if err != nil {
		return
	}

	streamUnified := &StreamUnified{
		Value: key,
	}
	buffer, err = proto.Marshal(streamUnified)
	if err != nil {
		return
	}

	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(env.GetString("server.proxied")).
		POST("https://api2.cursor.sh/aiserver.v1.ChatService/StreamUnifiedChatWithToolsSSE").
		Header("authorization", "Bearer "+cookie).
		Header("content-type", "application/connect+proto").
		Header("connect-accept-encoding", "gzip").
		Header("connect-content-encoding", "gzip").
		Header("connect-protocol-version", "1").
		Header("traceparent", "00-"+strings.ReplaceAll(uuid.NewString(), "-", "")+"-"+common.Hex(16)+"-00").
		Header("user-agent", "connect-es/1.6.1").
		Header("x-amzn-trace-id", "Root="+uuid.NewString()).
		Header("x-client-key", genClientKey(ctx.GetString("token"))).
		Header("x-cursor-checksum", genChecksum(ctx, env)).
		Header("x-cursor-client-version", "0.45.11").
		Header("x-cursor-timezone", "Asia/Shanghai").
		Header("x-ghost-mode", "false").
		Header("x-request-id", uuid.NewString()).
		Header("x-session-id", uuid.NewString()).
		Header("host", "api2.cursor.sh").
		Header("Connection", "close").
		Header("Transfer-Encoding", "chunked").
		Bytes(buffer).
		DoC(emit.Status(http.StatusOK), emit.IsPROTO)
	return
}

func convertRequest(completion model.Completion) (buffer []byte, err error) {
	messages := stream.Map(stream.OfSlice(completion.Messages), func(message model.Keyv[interface{}]) *ChatMessage_Content_Message {
		return &ChatMessage_Content_Message{
			Uid:   uuid.NewString(),
			Role:  elseOf[uint32](message.Is("role", "user"), 1, 2),
			Value: message.GetString("content"),
		}
	}).ToSlice()
	message := &ChatMessage{
		Content: &ChatMessage_Content{
			Messages: messages,
			Model: &ChatMessage_Content_Model{
				Value: completion.Model[7:],
			},
			Uid: uuid.NewString(),
			Info: &ChatMessage_Content_Info{
				Os:      "darwin",
				Arch:    "x64",
				Bash:    "/bin/zsh",
				Version: "22.2.0",
				Date:    time.Now().Format("2006-01-02T15:04:05.000Z"),
			},
		},
	}

	protoBytes, err := proto.Marshal(message)
	if err != nil {
		return
	}

	//header := int32ToBytes(0, len(protoBytes))
	//buffer = append(header, protoBytes...)

	buffer = protoBytes
	return
}

func checkUsage(ctx *gin.Context, env *env.Environment, max int) (count int, err error) {
	var (
		cookie = ctx.GetString("token")
	)
	cookie, err = url.QueryUnescape(cookie)
	if err != nil {
		return
	}

	user := ""
	if strings.Contains(cookie, "::") {
		user = strings.Split(cookie, "::")[0]
	}
	response, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(env.GetString("server.proxied")).
		GET("https://www.cursor.com/api/usage").
		Query("user", user).
		Header("cookie", "WorkosCursorSessionToken="+cookie).
		Header("referer", "https://www.cursor.com/settings").
		Header("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15").
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}
	defer response.Body.Close()
	obj, err := emit.ToMap(response)
	if err != nil {
		return
	}

	if som, ok := obj["startOfMonth"]; ok {
		t, e := time.Parse("2006-01-02T15:04:05.000Z", som.(string))
		if e != nil {
			logger.Error(e)
		} else {
			if t.Before(time.Now().Add(-(14 * 24 * time.Hour))) { // 超14天
				return
			}
		}
	}

	for k, v := range obj {
		if !strings.Contains(k, "gpt-") {
			continue
		}
		value, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		i := value["numRequests"].(float64)
		count += int(i)
	}

	count = max - count
	return
}

func genClientKey(token string) string {
	hex1 := sha256.Sum256([]byte(token + "--client-key"))
	return hex.EncodeToString(hex1[:])
}

func genChecksum(ctx *gin.Context, env *env.Environment) string {
	token := ctx.GetString("token")
	checksum := ctx.GetHeader("x-cursor-checksum")

	if checksum == "" {
		checksum = env.GetString("cursor.checksum")
		if strings.HasPrefix(checksum, "http") {
			cacheManager := cache.CursorCacheManager()
			value, err := cacheManager.GetValue(common.CalcHex(token))
			if err != nil {
				logger.Error(err)
				return ""
			}
			if value != "" {
				return value
			}

			response, err := emit.ClientBuilder(common.HTTPClient).GET(checksum).
				DoC(emit.Status(http.StatusOK), emit.IsTEXT)
			if err != nil {
				logger.Error(err)
				return ""
			}
			checksum = emit.TextResponse(response)
			response.Body.Close()

			_ = cacheManager.SetWithExpiration(common.CalcHex(token), checksum, 30*time.Minute) // 缓存30分钟
			return checksum
		}
	}

	if checksum == "" {
		// 不采用全局设备码方式，而是用cookie产生。更换时仅需要重新抓取新的WorkosCursorSessionToken即可
		salt := strings.Split(token, ".")
		calc := func(data []byte) {
			var t byte = 165
			for i := range data {
				data[i] = (data[i] ^ t) + byte(i)
				t = data[i]
			}
		}

		// 对时间检验了
		t := time.Now()
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 30*(t.Minute()/30), 0, 0, t.Location()) // 每个半小时轮换一次
		timestamp := int64(math.Floor(float64(t.UnixMilli()) / 1e6))
		data := []byte{
			byte((timestamp >> 8) & 0xff),
			byte(timestamp & 0xff),
			byte((timestamp >> 24) & 0xff),
			byte((timestamp >> 16) & 0xff),
			byte((timestamp >> 8) & 0xff),
			byte(timestamp & 0xff),
		}
		calc(data)
		hex1 := sha256.Sum256([]byte(salt[1]))
		hex2 := sha256.Sum256([]byte(token))
		// 前面的字符生成存在问题，先硬编码
		// woc , 粗心大意呀
		checksum = fmt.Sprintf("%s%s/%s", base64.RawStdEncoding.EncodeToString(data), hex.EncodeToString(hex1[:]), hex.EncodeToString(hex2[:]))
	}
	return checksum
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
