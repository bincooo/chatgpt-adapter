package common

import (
	"context"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"net/http"
	"time"
)

const baseURL = "https://bigjpg.com/api/task/"

// 图片放大api, 一个key每月有30次使用机会
//
//	url: 原图url
//	key: api-key
//	style: 参数有 'art', 'photo' 分别表示 '卡通插画', '照片'
//	x2: 参数有 '1', '2', '3', '4' 分别表示 2x, 4x, 8x, 16x
func magnify(ctx context.Context, url, key, style, x2 string) (string, error) {
	if style == "" {
		style = "art"
	}
	if x2 == "" {
		x2 = "1"
	}
	payload := map[string]interface{}{
		"style": style,
		"noise": "3",
		"x2":    x2,
		"input": url,
	}

	response, err := emit.ClientBuilder(nil).
		Context(ctx).
		URL(baseURL).
		Method(http.MethodPost).
		JHeader().
		Header("X-API-KEY", key).
		Body(payload).
		DoS(http.StatusOK)
	if err != nil {
		return "", err
	}

	if err = emit.ToObject(response, &payload); err != nil {
		return "", err
	}

	var taskId string
	if tid, ok := payload["tid"]; ok {
		taskId = tid.(string)
	} else {
		if status, k := payload["status"]; k {
			return "", fmt.Errorf("hd-api: %s", status)
		}
		return "", errors.New("hd-api: fetch task failed")
	}

	retry := 20
	for {
		if retry < 0 {
			return "", errors.New("hd-api: poll failed")
		}
		retry--

		response, err = emit.ClientBuilder(nil).
			Context(ctx).
			URL(baseURL + taskId).
			DoS(http.StatusOK)
		if err != nil {
			return "", err
		}

		if err = emit.ToObject(response, &payload); err != nil {
			return "", err
		}

		if data, ok := payload[taskId]; ok {
			m := data.(map[string]interface{})
			if status, k := m["status"]; k {
				if status == "success" {
					return m["url"].(string), nil
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
}
