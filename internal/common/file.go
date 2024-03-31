package common

import (
	"encoding/base64"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
)

func CreateBase64Image(base64Encoding, suffix string) (file string, err error) {
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
			logrus.Error("CreateBase64Image: ", err)
			return "", err
		}
	}

	tempFile, err := os.CreateTemp("tmp", "image-*."+suffix)
	if err != nil {
		logrus.Error("CreateBase64Image: ", err)
		return "", err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(dec)
	if err != nil {
		logrus.Error("CreateBase64Image: ", err)
		return "", err
	}

	return tempFile.Name(), nil
}

func DownloadImage(proxies, weburl, suffix string) (file string, err error) {
	client, err := NewHttpClient(proxies)
	if err != nil {
		logrus.Error("DownloadImage: ", err)
		return "", err
	}

	response, err := client.Get(weburl)
	if err != nil {
		logrus.Error("DownloadImage: ", err)
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		logrus.Error("DownloadImage: ", response.Status)
		return "", errors.New(response.Status)
	}

	dec, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.Error("DownloadImage: ", err)
		return "", err
	}

	_, err = os.Stat("tmp")
	if os.IsNotExist(err) {
		err = os.Mkdir("tmp", os.ModePerm)
		if err != nil {
			logrus.Error("DownloadImage: ", err)
			return "", err
		}
	}

	tempFile, err := os.CreateTemp("tmp", "image-*."+suffix)
	if err != nil {
		logrus.Error("DownloadImage: ", err)
		return "", err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(dec)
	if err != nil {
		logrus.Error("DownloadImage: ", err)
		return "", err
	}

	return tempFile.Name(), nil
}
