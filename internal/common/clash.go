package common

import (
	"context"
	"github.com/bincooo/chatgpt-adapter/logger"
	"github.com/bincooo/chatgpt-adapter/pkg"
	"github.com/bincooo/emit.io"
	"net/http"
	"sync"
	"time"
)

var (
	clashNames []string
	clashPos   int
	clashOnce  sync.Once
)

func init() {
	// 将初始化时机转移，而不是包引用则执行
	AddInitialized(func() {
		names := pkg.Config.GetStringSlice("clash.names")
		if len(names) == 0 {
			return
		}
		clashNames = names
		clashPos = 0
	})
}

func ChangeClashIP() {
	nameL := len(clashNames)
	if nameL == 0 {
		logger.Info("clash配置未开启")
		return
	}

	url := pkg.Config.GetString("clash.url")
	clashOnce.Do(func() {
		clashPos++
		if clashPos >= nameL {
			clashPos = 0
		}

		str := clashNames[clashPos]
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		_, err := emit.ClientBuilder().
			PUT(url).
			Context(ctx).
			JHeader().
			Body(map[string]string{"name": str}).
			DoS(http.StatusNoContent)
		if err != nil {
			logger.Error(err)
			return
		}
		logger.Infof("clash[%s]切换执行完毕", str)
	})
}
