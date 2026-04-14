package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0"
)

func Upload(chunk []byte, suffix string) (imageUrl, mimetype string, err error) {
	w := &bytes.Buffer{}
	writer := multipart.NewWriter(w)
	mimetype = mime.TypeByExtension("." + suffix)
	timestamp := time.Now().UnixNano()
	_ = writer.WriteField("json", "true")

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", multipart.FileContentDisposition("file_1", fmt.Sprintf("%d.%s", timestamp, suffix)))
	h.Set("Content-Type", mimetype)
	fw, _ := writer.CreatePart(h)
	_, err = fw.Write(chunk)
	if err != nil {
		return
	}

	err = writer.Close()
	if err != nil {
		return
	}

	request, err := http.NewRequest(http.MethodPost, "http://uploader.sh", w)
	if err != nil {
		return
	}

	request.Header.Set("content-type", writer.FormDataContentType())
	request.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	request.Header.Set("Sec-Ch-Ua-Platform", "\"macOS\"")
	request.Header.Set("user-agent", userAgent)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("upload failed with status code %d", response.StatusCode)
		return
	}

	chunk, err = io.ReadAll(response.Body)
	if err != nil {
		return
	}

	var result map[string]interface{}
	err = json.Unmarshal(chunk, &result)
	if err != nil {
		return
	}

	imageUrl = result["file_1"].(map[string]interface{})["url"].(string)
	return
}
