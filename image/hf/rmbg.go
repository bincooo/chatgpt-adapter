package hf

import (
	"bypass/tool"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/xllm-go/g/model"
)

func removeBackground(ctx *model.Ctx, path string) (value string, err error) {
	var (
		baseUrl = "https://briaai-bria-rmbg-2-0.hf.space"
	)

	var buf []byte
	if strings.HasPrefix(path, "http") {
		buf, err = tool.Download(http.DefaultClient, path, map[string]string{
			"origin":  "https://huggingface.co",
			"referer": baseUrl + "/?__theme=light",
		})
	} else {
		buf, err = os.ReadFile(path)
	}
	if err != nil {
		return
	}

	var buffer bytes.Buffer
	w := multipart.NewWriter(&buffer)
	fw, _ := w.CreateFormFile("files", filepath.Base(path))
	_, err = io.Copy(fw, bytes.NewBuffer(buf))
	if err != nil {
		return "", err
	}
	_ = w.Close()

	sessionHash := hash()
	request, err := http.NewRequest(http.MethodPost, baseUrl+"/upload?upload_id="+sessionHash, bytes.NewReader(buffer.Bytes()))
	if err != nil {
		return
	}

	request.Header.Set("content-type", w.FormDataContentType())
	request.Header.Set("referer", baseUrl+"/?__theme=light")
	request.Header.Set("user-agent", userAgent)
	request.Header.Set("accept-language", "en-US,en;q=0.9")
	request.Header.Set("origin", baseUrl)
	response, err := Do(http.DefaultClient.Do(request))(isStatus(http.StatusOK), isJson)
	if err != nil {
		return
	}

	var slice []string

	err = ToObject(response, &slice)
	_ = response.Body.Close()
	if err != nil {
		return
	}

	data := map[string]interface{}{
		"fn_index":     0,
		"trigger_id":   13,
		"session_hash": sessionHash,
		"data": []interface{}{
			map[string]interface{}{
				"path":      slice[0],
				"url":       baseUrl + slice[0],
				"orig_name": filepath.Base(slice[0]),
				"size":      len(buf),
				"mime_type": mimetype.Detect(buf).String(), //"image/png",
				"meta": map[string]string{
					"_type": "gradio.FileData",
				},
			},
		},
	}
	buf, err = json.Marshal(data)
	if err != nil {
		return
	}

	request, err = http.NewRequest(http.MethodPost, baseUrl+"/queue/join", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	request.Header.Set("Origin", baseUrl)
	request.Header.Set("Referer", baseUrl+"/?__theme=light")
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	response, err = Do(http.DefaultClient.Do(request))(isStatus(http.StatusOK), isJson)
	if err != nil {
		return
	}
	_ = response.Body.Close()

	request, err = http.NewRequest(http.MethodGet, baseUrl+"/queue/data?session_hash="+sessionHash, nil)
	if err != nil {
		return
	}

	request.Header.Set("Origin", baseUrl)
	request.Header.Set("Referer", baseUrl+"/?__theme=light")
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	response, err = Do(http.DefaultClient.Do(request))()
	if err != nil {
		return
	}

	defer response.Body.Close()
	scanner, err := newScanner(ctx.Context(), response)
	if err != nil {
		return
	}

	scanner.Event("process_completed", func(j JoinEvent) (_ interface{}) {
		if len(j.Output.Data) == 0 {
			scanner.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		i := j.Output.Data[len(j.Output.Data)-1]
		dict := i.(map[string]interface{})
		value = dict["url"].(string)
		return
	})

	err = scanner.Do()
	return
}
