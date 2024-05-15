package common

import (
	"context"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	emits "github.com/bincooo/gio.emits"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

var (
	clashNames []string
	clashPos   int
	clashOnce  sync.Once
)

func clashInit() {
	names := pkg.Config.GetStringSlice("clash.names")
	if len(names) == 0 {
		return
	}
	clashNames = names
	clashPos = 0
}

func ChangeClashIP() {
	nameL := len(clashNames)
	if nameL == 0 {
		logrus.Info("clash配置未开启")
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

		_, err := emits.ClientBuilder().
			PUT(url).
			Context(ctx).
			JHeader().
			Body(map[string]string{"name": str}).
			DoS(http.StatusNoContent)
		if err != nil {
			logrus.Error(err)
			return
		}
		logrus.Infof("clash[%s]切换执行完毕", str)
	})
}
