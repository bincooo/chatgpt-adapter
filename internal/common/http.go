package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type R struct {
	url     string
	method  string
	proxies string
	headers map[string]string
	query   []string
	bytes   []byte
	err     error
	ctx     context.Context
}

func ClientBuilder() *R {
	return &R{
		method:  http.MethodGet,
		query:   make([]string, 0),
		headers: make(map[string]string),
	}
}

func (r *R) URL(url string) *R {
	r.url = url
	return r
}

func (r *R) Method(method string) *R {
	r.method = method
	return r
}

func (r *R) GET(url string) *R {
	r.url = url
	r.method = http.MethodGet
	return r
}

func (r *R) POST(url string) *R {
	r.url = url
	r.method = http.MethodPost
	return r
}

func (r *R) PUT(url string) *R {
	r.url = url
	r.method = http.MethodPut
	return r
}

func (r *R) DELETE(url string) *R {
	r.url = url
	r.method = http.MethodDelete
	return r
}

func (r *R) Proxies(proxies string) *R {
	r.proxies = proxies
	return r
}

func (r *R) Context(ctx context.Context) *R {
	r.ctx = ctx
	return r
}

func (r *R) JsonHeader() *R {
	r.headers["content-type"] = "application/json"
	return r
}

func (r *R) Header(key, value string) *R {
	r.headers[key] = value
	return r
}

func (r *R) Query(key, value string) *R {
	r.query = append(r.query, fmt.Sprintf("%s=%s", key, value))
	return r
}

func (r *R) SetBody(payload interface{}) *R {
	if r.err != nil {
		return r
	}
	r.bytes, r.err = json.Marshal(payload)
	return r
}

func (r *R) SetBytes(data []byte) *R {
	r.bytes = data
	return r
}

func (r *R) DoWith(status int) (*http.Response, error) {
	response, err := r.Do()
	if err != nil {
		return nil, err
	}

	if response.StatusCode != status {
		return nil, errors.New(response.Status)
	}

	return response, nil
}

func (r *R) Do() (*http.Response, error) {
	if r.err != nil {
		return nil, r.err
	}

	if r.url == "" {
		return nil, errors.New("url cannot be nil, please execute func URL(url string)")
	}

	c, err := client(r.proxies)
	if err != nil {
		return nil, err
	}

	query := ""
	if len(r.query) > 0 {
		var slice []string
		for _, value := range r.query {
			slice = append(slice, value)
		}
		query = "?" + strings.Join(slice, "&")
	}
	request, err := http.NewRequest(r.method, r.url+query, bytes.NewBuffer(r.bytes))
	if err != nil {
		return nil, err
	}

	h := request.Header
	for k, v := range r.headers {
		h.Add(k, v)
	}

	if r.ctx != nil {
		request = request.WithContext(r.ctx)
	}

	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func client(proxies string) (*http.Client, error) {
	c := http.DefaultClient
	if proxies != "" {
		proxiesUrl, err := url.Parse(proxies)
		if err != nil {
			return nil, err
		}

		if proxiesUrl.Scheme == "http" || proxiesUrl.Scheme == "https" {
			c = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxiesUrl),
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}

		// socks5://127.0.0.1:7890
		if proxiesUrl.Scheme == "socks5" {
			c = &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						dialer, e := proxy.SOCKS5("tcp", proxiesUrl.Host, nil, proxy.Direct)
						if e != nil {
							return nil, e
						}
						return dialer.Dial(network, addr)
					},
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
	}

	return c, nil
}

func ToObject(response *http.Response, obj interface{}) error {
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, obj); err != nil {
		return err
	}

	return nil
}

func GetCookie(response *http.Response, key string) string {
	cookie := response.Header.Get("set-cookie")
	if !strings.HasPrefix(cookie, key+"=") {
		return ""
	}

	cookie = strings.TrimPrefix(cookie, key+"=")
	cos := strings.Split(cookie, "; ")
	if len(cos) > 0 {
		return cos[0]
	}

	return ""
}

// 获取随机ip
func RandomIp() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ip2Int := func(ip string) int {
		slice := strings.Split(ip, ".")
		result := 0
		atoi, _ := strconv.Atoi(slice[0])
		result += atoi << 24
		atoi, _ = strconv.Atoi(slice[1])
		result += atoi << 16
		atoi, _ = strconv.Atoi(slice[2])
		result += atoi << 8
		atoi, _ = strconv.Atoi(slice[2])
		result += atoi
		return result
	}

	int2Ip := func(num int) (result string) {
		result += strconv.Itoa(num>>24&255) + "."
		result += strconv.Itoa(num>>16&255) + "."
		result += strconv.Itoa(num>>8&255) + "."
		result += strconv.Itoa(num & 255)
		return
	}

	randIndex := r.Intn(len(IP_RANGE))
	startIPInt := ip2Int(IP_RANGE[randIndex][0])
	endIPInt := ip2Int(IP_RANGE[randIndex][1])

	newIpInt := r.Intn(endIPInt-startIPInt) + startIPInt
	return int2Ip(newIpInt)
}

var IP_RANGE = [][]string{
	{"4.150.64.0", "4.150.127.255"},      // Azure Cloud EastUS2 16382
	{"4.152.0.0", "4.153.255.255"},       // Azure Cloud EastUS2 131070
	{"13.68.0.0", "13.68.127.255"},       // Azure Cloud EastUS2 32766
	{"13.104.216.0", "13.104.216.255"},   // Azure EastUS2 256
	{"20.1.128.0", "20.1.255.255"},       // Azure Cloud EastUS2 32766
	{"20.7.0.0", "20.7.255.255"},         // Azure Cloud EastUS2 65534
	{"20.22.0.0", "20.22.255.255"},       // Azure Cloud EastUS2 65534
	{"40.84.0.0", "40.84.127.255"},       // Azure Cloud EastUS2 32766
	{"40.123.0.0", "40.123.127.255"},     // Azure Cloud EastUS2 32766
	{"4.214.0.0", "4.215.255.255"},       // Azure Cloud JapanEast 131070
	{"4.241.0.0", "4.241.255.255"},       // Azure Cloud JapanEast 65534
	{"40.115.128.0", "40.115.255.255"},   // Azure Cloud JapanEast 32766
	{"52.140.192.0", "52.140.255.255"},   // Azure Cloud JapanEast 16382
	{"104.41.160.0", "104.41.191.255"},   // Azure Cloud JapanEast 8190
	{"138.91.0.0", "138.91.15.255"},      // Azure Cloud JapanEast 4094
	{"151.206.65.0", "151.206.79.255"},   // Azure Cloud JapanEast 256
	{"191.237.240.0", "191.237.241.255"}, // Azure Cloud JapanEast 512
	{"4.208.0.0", "4.209.255.255"},       // Azure Cloud NorthEurope 131070
	{"52.169.0.0", "52.169.255.255"},     // Azure Cloud NorthEurope 65534
	{"68.219.0.0", "68.219.127.255"},     // Azure Cloud NorthEurope 32766
	{"65.52.64.0", "65.52.79.255"},       // Azure Cloud NorthEurope 4094
	{"98.71.0.0", "98.71.127.255"},       // Azure Cloud NorthEurope 32766
	{"74.234.0.0", "74.234.127.255"},     // Azure Cloud NorthEurope 32766
	{"4.151.0.0", "4.151.255.255"},       // Azure Cloud SouthCentralUS 65534
	{"13.84.0.0", "13.85.255.255"},       // Azure Cloud SouthCentralUS 131070
	{"4.255.128.0", "4.255.255.255"},     // Azure Cloud WestCentralUS 32766
	{"13.78.128.0", "13.78.255.255"},     // Azure Cloud WestCentralUS 32766
	{"4.175.0.0", "4.175.255.255"},       // Azure Cloud WestEurope 65534
	{"13.80.0.0", "13.81.255.255"},       // Azure Cloud WestEurope 131070
	{"20.73.0.0", "20.73.255.255"},       // Azure Cloud WestEurope 65534
}
