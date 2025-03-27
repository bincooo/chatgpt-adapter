package common

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"math/rand"
	"time"
	"unsafe"

	"chatgpt-adapter/core/logger"
)

type ref struct {
	rtype unsafe.Pointer
	data  unsafe.Pointer
}

func IsNIL(obj interface{}) bool {
	return obj == nil || unpackEFace(obj).data == nil
}

func Hex(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	bytes := make([]rune, n)
	for i := range bytes {
		bytes[i] = runes[r.Intn(len(runes))]
	}
	return string(bytes)
}

func RandInt(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var runes = []rune("1234567890")
	bytes := make([]rune, n)
	for i := range bytes {
		bytes[i] = runes[r.Intn(len(runes))]
	}
	return string(bytes)
}

func CalcHex(str string) string {
	h := sha1.New()
	if _, err := io.WriteString(h, str); err != nil {
		logger.Error(err)
		return "-1"
	}
	return hex.EncodeToString(h.Sum(nil))
}

func isSlice(o interface{}) (ok bool) {
	_, ok = o.([]interface{})
	return
}

func unpackEFace(obj interface{}) *ref {
	return (*ref)(unsafe.Pointer(&obj))
}

func ips(ips ...string) func() []string {
	return func() []string { return ips }
}
