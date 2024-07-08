package common

import (
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"github.com/bincooo/emit.io"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	initFunctions = make([]func(), 0)
	exitFunctions = make([]func(), 0)
)

func InitCommon() {
	for _, apply := range initFunctions {
		apply()
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func(ch chan os.Signal) {
		<-ch
		for _, apply := range exitFunctions {
			apply()
		}
		os.Exit(0)
	}(ch)
}

func AddInitialized(apply func()) {
	initFunctions = append(initFunctions, apply)
}

func AddExited(apply func()) {
	exitFunctions = append(exitFunctions, apply)
}

func GetIdleConnectOption() *emit.ConnectOption {
	opts := pkg.Config.GetStringMap("server-conn")
	var option emit.ConnectOption
	if value, ok := opts["idleconntimeout"]; ok {
		connTimeout, o := value.(int)
		if o {
			if connTimeout > 0 {
				option.IdleConnTimeout = time.Duration(connTimeout) * time.Second
			}
		} else {
			logger.Warnf("read idleConnTimeout error: %v", value)
		}
	}

	if value, ok := opts["responseheadertimeout"]; ok {
		connTimeout, o := value.(int)
		if o {
			if connTimeout > 0 {
				option.ResponseHeaderTimeout = time.Duration(connTimeout) * time.Second
			}
		} else {
			logger.Warnf("read responseHeaderTimeout error: %v", value)
		}
	}

	if value, ok := opts["expectcontinuetimeout"]; ok {
		connTimeout, o := value.(int)
		if o {
			if connTimeout > 0 {
				option.ExpectContinueTimeout = time.Duration(connTimeout) * time.Second
			}
		} else {
			logger.Warnf("read expectContinueTimeout error: %v", value)
		}
	}

	option.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &option
}

// 删除子元素
func Remove[T comparable](slice []T, t T) ([]T, int) {
	return RemoveFor(slice, func(item T) bool {
		return item == t
	})
}

// 删除子元素, condition：自定义判断规则
func RemoveFor[T comparable](slice []T, condition func(item T) bool) ([]T, int) {
	if len(slice) == 0 {
		return slice, -1
	}

	for idx := 0; idx < len(slice); idx++ {
		if condition(slice[idx]) {
			slice = append(slice[:idx], slice[idx+1:]...)
			return slice, idx
		}
	}

	return slice, -1
}

// 判断切片是否包含子元素
func Contains[T comparable](slice []T, t T) bool {
	return ContainFor(slice, func(item T) bool {
		return item == t
	})
}

// 判断切片是否包含子元素， condition：自定义判断规则
func ContainFor[T comparable](slice []T, condition func(item T) bool) bool {
	if len(slice) == 0 {
		return false
	}

	for idx := 0; idx < len(slice); idx++ {
		if condition(slice[idx]) {
			return true
		}
	}
	return false
}

func RandString(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	bytes := make([]rune, n)
	for i := range bytes {
		bytes[i] = runes[r.Intn(len(runes))]
	}
	return string(bytes)
}

func HashString(str string) string {
	h := sha1.New()
	if _, err := io.WriteString(h, str); err != nil {
		logger.Error(err)
		return "-1"
	}
	return hex.EncodeToString(h.Sum(nil))
}
