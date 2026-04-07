package hf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func Do(response *http.Response, err error) func(funSlice ...func(*http.Response) error) (*http.Response, error) {
	return func(funSlice ...func(*http.Response) error) (*http.Response, error) {
		if err != nil {
			return response, err
		}

		for _, condition := range funSlice {
			if err = condition(response); err != nil {
				if response != nil {
					_ = response.Body.Close()
				}
				return response, err
			}
		}

		return response, nil
	}
}

func isJson(response *http.Response) error {
	return ist(response, "application/json")
}

func isPlain(response *http.Response) error {
	return ist(response, "text/plain")
}

func isHtml(response *http.Response) error {
	return ist(response, "text/html")
}

func isStream(response *http.Response) error {
	return ist(response, "text/event-stream", "application/stream")
}

func isProto(response *http.Response) error {
	return ist(response, "application/connect+proto", "application/proto")
}

func isStatus(status int) func(response *http.Response) error {
	return func(response *http.Response) error {
		if response == nil {
			return errors.New("response is nil")
		}
		if response.StatusCode != status {
			msg := ""
			if isJ(response.Header) {
				msg = textResponse(response)
			}
			_ = response.Body.Close()
			return fmt.Errorf("[%s] %s", response.Status, msg)
		}
		return nil
	}
}

func ist(response *http.Response, ts ...string) error {
	if response == nil {
		return errors.New("response is nil")
	}

	h := response.Header
	for _, t := range ts {
		if strings.Contains(h.Get("content-type"), t) {
			return nil
		}
	}

	_ = response.Body.Close()
	if isJ(response.Header) {
		return fmt.Errorf("response is not [ %s ], %s", ts, textResponse(response))
	}

	return fmt.Errorf("response is not [ %s ]", ts)
}

func isJ(header http.Header) bool {
	if header == nil {
		return false
	}
	return strings.Contains(header.Get("content-type"), "application/json")
}

func textResponse(response *http.Response) (value string) {
	if response == nil {
		return
	}
	chunk, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	return string(chunk)
}

func ToObject(response *http.Response, obj interface{}) (err error) {
	var data []byte
	data, err = io.ReadAll(response.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, obj)
	return
}

func ToMap(response *http.Response) (obj map[string]interface{}, err error) {
	err = ToObject(response, &obj)
	return
}

func ToSlice(response *http.Response) (slice []map[string]interface{}, err error) {
	err = ToObject(response, &slice)
	return
}
