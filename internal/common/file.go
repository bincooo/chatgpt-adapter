package common

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
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
		_ = os.Mkdir("tmp", os.ModePerm)
	}

	tempFile, err := os.CreateTemp("tmp", "image-*."+suffix)
	if err != nil {
		return "", err
	}

	_, err = tempFile.Write(decode)
	if err != nil {
		return "", err
	}

	file = tempFile.Name()
	return file, nil
}

func UploadCatboxFile(proxies, file string) (string, error) {
	client, err := NewHttpClient(proxies)
	if err != nil {
		return "", err
	}

	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)

	if !strings.HasPrefix(file, "http") {
		suffix := "bin"
		index := strings.LastIndex(file, ".")
		if index > 0 {
			suffix = file[index+1:]
		}

		w.WriteField("reqtype", "fileupload")
		w.WriteField("userhash", "")
		fw, e := w.CreateFormFile("fileToUpload", "1."+suffix)
		if e != nil {
			return "", e
		}

		fi, e := os.Open(file)
		if e != nil {
			return "", e
		}

		_, _ = io.Copy(fw, fi)
		w.Close()
	} else {
		w.WriteField("reqtype", "urlupload")
		w.WriteField("userhash", "")
		w.WriteField("url", file)
		w.Close()
	}

	response, e := client.Post("https://catbox.moe/user/api.php", w.FormDataContentType(), body)
	if e != nil {
		return "", e
	}

	if response.StatusCode != http.StatusOK {
		return "", errors.New(response.Status)
	}

	data, e := io.ReadAll(response.Body)
	if e != nil {
		return "", e
	}
	return string(data), nil
}
