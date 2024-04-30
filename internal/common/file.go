package common

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/bincooo/gio.emits/common"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var cached []cache = nil

type cache struct {
	time.Time
	key     string
	disable bool
}

func fileInit() {
	magnify := pkg.Config.GetStringSlice("magnify")
	if len(magnify) == 0 {
		return
	}

	for _, key := range magnify {
		cached = append(cached, cache{
			key:     key,
			disable: false,
		})
	}
}

func HasMfy() bool {
	return len(cached) > 0
}

func Magnify(ctx context.Context, url string) (jpgurl string, err error) {
	for _, c := range cached {
		if c.disable && c.After(time.Now()) {
			continue
		}

		jpgurl, err = magnify(ctx, url, c.key, "art", "1")
		if err != nil {
			c.disable = true
			c.Add(5 * time.Minute) // 5m 内不参与轮询
			continue
		}

		c.disable = false
		return jpgurl, nil
	}

	if err != nil {
		return
	}

	return "", errors.New("poll failed")
}

func SaveBase64(base64Encoding, suffix string) (file string, err error) {
	index := strings.Index(base64Encoding, ",")
	base64Encoding = base64Encoding[index+1:]
	dec, err := base64.StdEncoding.DecodeString(base64Encoding)
	if err != nil {
		return "", err
	}

	_, err = os.Stat("tmp")
	if os.IsNotExist(err) {
		err = os.Mkdir("tmp", os.ModePerm)
		if err != nil {
			logrus.Error("save base64 failed: ", err)
			return "", err
		}
	}

	tempFile, err := os.CreateTemp("tmp", "*."+suffix)
	if err != nil {
		logrus.Error("save base64 failed: ", err)
		return "", err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(dec)
	if err != nil {
		logrus.Error("save base64 failed: ", err)
		return "", err
	}

	return tempFile.Name(), nil
}

func Download(proxies, url, suffix string) (file string, err error) {
	response, err := common.ClientBuilder().
		Proxies(proxies).
		URL(url).
		Do()
	if err != nil {
		logrus.Error("download failed: ", err)
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		logrus.Error("download failed: ", response.Status)
		return "", errors.New(response.Status)
	}

	dec, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.Error("download failed: ", err)
		return "", err
	}

	_, err = os.Stat("tmp")
	if os.IsNotExist(err) {
		err = os.Mkdir("tmp", os.ModePerm)
		if err != nil {
			logrus.Error("download failed: ", err)
			return "", err
		}
	}

	tempFile, err := os.CreateTemp("tmp", "*."+suffix)
	if err != nil {
		logrus.Error("download failed: ", err)
		return "", err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(dec)
	if err != nil {
		logrus.Error("download failed: ", err)
		return "", err
	}

	return tempFile.Name(), nil
}
