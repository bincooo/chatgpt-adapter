package tool

import (
	"fmt"
	"io"
	"net/http"
)

func Download(url string, header map[string]string) (buffer []byte, err error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	request.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	request.Header.Set("Sec-Ch-Ua-Platform", "\"macOS\"")
	request.Header.Set("Sec-Fetch-Dest", "image")
	request.Header.Set("accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	for k, v := range header {
		request.Header.Set(k, v)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("download failed with status code %d", response.StatusCode)
		return
	}

	buffer, err = io.ReadAll(response.Body)
	return
}
