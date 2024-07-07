package common

import (
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"errors"
	"github.com/bincooo/emit.io"
	"net/http"
	"sync"
	"time"
)

var (
	poster    = make(map[string]int)
	clashOnce = make(map[string]*sync.Once)
)

func ChangeClashIP(key string) error {
	clashNames := pkg.Config.GetStringSlice("clash." + key + ".names")
	nameL := len(clashNames)
	if nameL == 0 {
		return nil
	}

	url := pkg.Config.GetString("clash." + key + ".url")
	if url == "" {
		return errors.New("clash未配置: " + key)
	}

	once, ok := clashOnce[key]
	if !ok {
		once = new(sync.Once)
		clashOnce[key] = once
	}

	once.Do(func() {
		pos, o := poster[key]
		if !o {
			pos = 0
		}

		if pos >= nameL {
			pos = 0
		}

		str := clashNames[pos]
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		_, err := emit.ClientBuilder(nil).
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
		poster[key] = pos + 1
		delete(clashOnce, key)
	})
	return nil
}
