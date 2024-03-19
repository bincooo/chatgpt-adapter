package common

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
)

func CreateBase64Image(base64Encoding, suffix string) (file string, err error) {
	index := strings.Index(base64Encoding, ",")
	base64Encoding = base64Encoding[index+1:]
	decode, err := base64.StdEncoding.DecodeString(base64Encoding)
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

	_, err = tempFile.Write(decode)
	if err != nil {
		logrus.Error("CreateBase64Image: ", err)
		return "", err
	}

	return tempFile.Name(), nil
}

func UploadCatboxFile(proxies, file string) (string, error) {
	client, err := NewHttpClient(proxies)
	if err != nil {
		logrus.Error("UploadCatboxFile: ", err)
		return "", err
	}

	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	defer w.Close()

	if !strings.HasPrefix(file, "http") {
		w.WriteField("reqtype", "fileupload")
		w.WriteField("userhash", "")
		fw, e := w.CreateFormFile("fileToUpload", path.Base(file))
		if e != nil {
			logrus.Error("UploadCatboxFile: ", e)
			return "", e
		}

		fi, e := os.Open(file)
		if e != nil {
			logrus.Error("UploadCatboxFile: ", e)
			return "", e
		}

		defer fi.Close()
		_, _ = io.Copy(fw, fi)

	} else {
		w.WriteField("reqtype", "urlupload")
		w.WriteField("userhash", "")
		w.WriteField("url", file)
	}

	response, e := client.Post("https://catbox.moe/user/api.php", w.FormDataContentType(), body)
	if e != nil {
		logrus.Error("UploadCatboxFile: ", e)
		return "", e
	}

	if response.StatusCode != http.StatusOK {
		logrus.Error("UploadCatboxFile: ", response.Status)
		return "", errors.New(response.Status)
	}

	data, e := io.ReadAll(response.Body)
	if e != nil {
		logrus.Error("UploadCatboxFile: ", e)
		return "", e
	}
	return string(data), nil
}
