package tool

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/xllm-go/g"
)

var (
	Sdk = g.Sdk()
)

func Upload(chunk []byte, suffix string) (imageUrl, mimetype string, err error) {
	baseUrl := "https://s30hxbqg-transfer.hf.space"

	mimetype = mime.TypeByExtension("." + suffix)
	timestamp := time.Now().UnixNano()

	if transfer := Sdk.Env().GetString("transfer"); transfer != "" {
		baseUrl = transfer
	}

	request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%d.%s", baseUrl, timestamp, suffix), bytes.NewBuffer(chunk))
	if err != nil {
		return
	}

	request.Header.Set("content-type", mimetype)
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

	u, err := url.Parse(string(chunk))
	if err != nil {
		return
	}

	u.Path = path.Join("get", u.Path)
	imageUrl = u.String()
	return
}
