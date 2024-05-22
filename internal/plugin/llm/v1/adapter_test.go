package v1

import (
	"bufio"
	"bytes"
	"github.com/bincooo/chatgpt-adapter/v2/pkg"
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
	_ "time/tzdata"
)

func TestGpt35(t *testing.T) {
	ctx := new(gin.Context)
	ctx.Request, _ = http.NewRequest("POST", "http://localhost:3000", nil)
	ctx.Set("proxies", "http://127.0.0.1:7890")
	response, err := fetchGpt35(ctx, pkg.ChatCompletion{
		Model: "text-davinci-002-render-sha",
		Messages: []pkg.Keyv[interface{}]{
			{
				"role":    "user",
				"content": "hi",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	echo(t, response)
}

func echo(t *testing.T, response *http.Response) {
	scanner := bufio.NewScanner(response.Body)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if eof {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for {
		if !scanner.Scan() {
			return
		}

		text := scanner.Text()
		t.Log(text)
	}
}
