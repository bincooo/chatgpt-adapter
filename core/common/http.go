package common

import (
	"crypto/tls"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/iocgo/sdk/env"
)

var (
	HTTPClient    *emit.Session
	NopHTTPClient *emit.Session
)

func init() {
	inited.AddInitialized(func(env *env.Environment) {
		var err error
		proxied := env.GetString("server.proxied")
		options := GetIdleConnectOptions(env)
		connTimeout := env.GetInt("server-conn.connTimeout")
		if connTimeout == 0 {
			connTimeout = 180
		}

		options = append(options, emit.Ja3Helper(emit.Echo{RandomTLSExtension: true, HelloID: profiles.Chrome_124}, connTimeout))
		HTTPClient, err = emit.NewSession(proxied, ips("127.0.0.1"), options...)
		if err != nil {
			logger.Fatal("Error initializing HTTPClient: ", err)
		}

		NopHTTPClient, err = emit.NewSession("", nil, options...)
		if err != nil {
			logger.Fatal("Error initializing HTTPClient: ", err)
		}
	})

	inited.AddInitialized(func(env *env.Environment) {
		if !env.GetBool("browser-less.enabled") {
			return
		}

		port := env.GetString("browser-less.port")
		if port == "" {
			logger.Fatal("please config browser-less.port to use")
		}

		proxied := env.GetString("server.proxied")
		Exec(port, proxied, os.Stdout, os.Stderr)
		inited.AddExited(Exit)
	})
}

func GetIdleConnectOptions(env *env.Environment) (options []emit.OptionHelper) {
	opts := env.GetStringMap("server-conn")
	if value, ok := opts["idleconntimeout"]; ok {
		timeout, o := value.(int)
		if o {
			if timeout > 0 {
				options = append(options, emit.IdleConnTimeoutHelper(time.Duration(timeout)*time.Second))
			}
		} else {
			logger.Warnf("read idleConnTimeout error: %v", value)
		}
	}

	if value, ok := opts["responseheadertimeout"]; ok {
		timeout, o := value.(int)
		if o {
			if timeout > 0 {
				options = append(options, emit.ResponseHeaderTimeoutHelper(time.Duration(timeout)*time.Second))
			}
		} else {
			logger.Warnf("read responseHeaderTimeout error: %v", value)
		}
	}

	if value, ok := opts["expectcontinuetimeout"]; ok {
		timeout, o := value.(int)
		if o {
			if timeout > 0 {
				options = append(options, emit.ExpectContinueTimeoutHelper(time.Duration(timeout)*time.Second))
			}
		} else {
			logger.Warnf("read expectContinueTimeout error: %v", value)
		}
	}

	options = append(options, emit.TLSConfigHelper(&tls.Config{InsecureSkipVerify: true}))
	return
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
		// Ja3(ja3).
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
		// session.IdleClose()
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

	return tempFile.Name()[4:], nil
}
