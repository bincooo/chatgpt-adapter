package utils

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type HttpRequest struct {
	client  *http.Client
	request *http.Request
	err     error
}

func NewHttp() *HttpRequest {
	return &HttpRequest{
		client: &http.Client{},
	}
}

func (req *HttpRequest) Timeout(timeout time.Duration) *HttpRequest {
	if req.err != nil {
		return req
	}
	req.client.Timeout = timeout
	return req
}

func (req *HttpRequest) GET(url string) *HttpRequest {
	return req.withRequest(url, http.MethodGet, nil)
}

func (req *HttpRequest) POST(url string, body io.Reader) *HttpRequest {
	return req.withRequest(url, http.MethodPost, body)
}

func (req *HttpRequest) ProxyHttp(host string, port int) *HttpRequest {
	return req.ProxyString("http://" + host + ":" + strconv.Itoa(port))
}

func (req *HttpRequest) ProxyHttps(host string, port int) *HttpRequest {
	return req.ProxyString("https://" + host + ":" + strconv.Itoa(port))
}

func (req *HttpRequest) ProxyString(uRL string) *HttpRequest {
	parser, err := url.Parse(uRL)
	if err != nil {
		req.err = err
		return req
	}
	if err != nil {
		req.err = err
		return req
	}
	req.client.Transport = &http.Transport{
		Proxy: http.ProxyURL(parser),
	}
	return req
}

func (req *HttpRequest) AddHeader(name string, value string) *HttpRequest {
	if req.err != nil {
		return req
	}
	if req.request == nil {
		req.err = errors.New("please initialize `Request` before executing here")
		return req
	}
	req.request.Header.Add(name, value)
	return req
}

func (req *HttpRequest) Build() (*http.Response, error) {
	if req.err != nil {
		return nil, req.err
	}
	if req.request == nil {
		return nil, errors.New("please initialize `Request` before executing here")
	}
	return req.client.Do(req.request)
}

func (req *HttpRequest) withRequest(url string, method string, body io.Reader) *HttpRequest {
	if req.err != nil {
		return req
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		req.err = err
		return req
	}
	req.request = request
	return req
}
