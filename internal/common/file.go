package common

import (
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/bincooo/emit.io"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var cached []cache = nil

const ja3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513-21,29-23-24,0"

type cache struct {
	time.Time
	key     string
	disable bool
}

func init() {
	// 将初始化时机转移，而不是包引用则执行
	AddInitialized(func() {
		m := pkg.Config.GetStringSlice("magnify")
		if len(m) == 0 {
			return
		}

		for _, key := range m {
			cached = append(cached, cache{
				key:     key,
				disable: false,
			})
		}
	})
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

	timePath := time.Now().Format("2006/01/02")
	_, err = os.Stat("tmp/" + timePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll("tmp/"+timePath, 0766)
		if err != nil {
			logger.Error("save base64 failed: ", err)
			return "", err
		}
	}

	tempFile, err := os.CreateTemp("tmp/"+timePath, "*."+suffix)
	if err != nil {
		logger.Error("save base64 failed: ", err)
		return "", err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(dec)
	if err != nil {
		logger.Error("save base64 failed: ", err)
		return "", err
	}

	return tempFile.Name(), nil
}

func Download(session *emit.Session, proxies, url, suffix string, header map[string]string) (file string, err error) {
	builder := emit.ClientBuilder(session).
		//Ja3(ja3).
		Proxies(proxies).
		GET(url).
		Header("Sec-Ch-Ua-Mobile", "?0").
		Header("Sec-Ch-Ua-Platform", "\"macOS\"").
		Header("Sec-Fetch-Dest", "image").
		Header("accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	for k, v := range header {
		builder.Header(k, v)
	}

	var response *http.Response
	responses := make([]*http.Response, 0)
	defer func() {
		for _, r := range responses {
			_ = r.Body.Close()
		}
		//session.IdleClose()
	}()

	retry := 3
label:
	retry--

	response, err = builder.DoS(http.StatusOK)
	if err != nil {
		if retry > 0 {
			time.Sleep(time.Second)
			goto label
		}
		return "", err
	}

	responses = append(responses, response)
	dec, err := io.ReadAll(response.Body)
	if err != nil {
		if retry > 0 {
			time.Sleep(time.Second)
			goto label
		}
		return "", err
	}

	timePath := time.Now().Format("2006/01/02")
	_, err = os.Stat("tmp/" + timePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll("tmp/"+timePath, 0766)
		if err != nil {
			return "", err
		}
	}

	tempFile, err := os.CreateTemp("tmp/"+timePath, "*."+suffix)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(dec)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func LoadImageMeta(url string) (mime string, data string, err error) {
	// base64
	if strings.HasPrefix(url, "data:image/") {
		pos := strings.Index(url, ";")
		if pos == -1 {
			err = errors.New("invalid base64 url")
			return
		}

		mime = url[5:pos]
		url = url[pos+1:]

		if !strings.HasPrefix(url, "base64,") {
			err = errors.New("invalid base64 url")
			return
		}
		data = url[7:]
		return
	}

	// url
	response, err := http.Get(url)
	if err != nil {
		return
	}

	defer response.Body.Close()
	dataBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	mime = response.Header.Get("content-type")
	data = base64.StdEncoding.EncodeToString(dataBytes)
	return
}

func CalcSHA256(buffer []byte) string {
	hasher := sha256.New()
	hasher.Write(buffer)
	return hex.EncodeToString(hasher.Sum(nil))
}
