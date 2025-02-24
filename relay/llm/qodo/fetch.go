package qodo

import (
	"chatgpt-adapter/core/cache"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"encoding/base64"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iocgo/sdk/env"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 Edg/133.0.0.0"
	// 输入检测对抗
	mapC = map[string]string{
		"是": "似",
		"的": "の",
		"人": "ren",
		"有": "you",
		"不": "bu",
		"上": "shang",
		"我": "窝",
		"他": "ta",
		"了": "le",
	}
)

type qodoRequest struct {
	MaxRemoteContext  int           `json:"max_remote_context"`
	RemoteContextTags []interface{} `json:"remote_context_tags"`
	MaxRepoContext    int           `json:"max_repo_context"`
	UserData          struct {
		InstallationId              string `json:"installation_id"`
		InstallationFingerprintUuid string `json:"installation_fingerprint_uuid"`
		EditorVersion               string `json:"editor_version"`
		ExtensionVersion            string `json:"extension_version"`
		OsPlatform                  string `json:"os_platform"`
		OsVersion                   string `json:"os_version"`
		EditorType                  string `json:"editor_type"`
	} `json:"user_data"`
	Task             string `json:"task"`
	ChatInput        string `json:"chat_input"`
	PreviousMessages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
		Command string `json:"command,omitempty"`
		Mode    string `json:"mode,omitempty"`
	} `json:"previous_messages"`
	UserContext       []interface{} `json:"user_context"`
	RepoContext       []interface{} `json:"repo_context"`
	CustomModel       string        `json:"custom_model"`
	SupportsArtifacts bool          `json:"supports_artifacts"`
}

func fetch(ctx *gin.Context, proxied string, request qodoRequest) (response *http.Response, err error) {
	token, err := genToken(ctx, env.Env)
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST("https://api.gen.qodo.ai/v2/chats/chat").
		JSONHeader().
		Header("Accept", "text/plain").
		Header("Host", "api.gen.qodo.ai").
		Header("User-Agent", "axios/1.7.9").
		Header("Accept-Encoding", "gzip, compress, deflate, br").
		Header("Authorization", "Bearer "+token).
		Header("Connection", "close").
		Body(request).
		DoS(http.StatusOK)
	return
}

func convertRequest(ctx *gin.Context, env *env.Environment, completion model.Completion) (request qodoRequest, err error) {
	chatInput := "hi"
	//contentBuffer := new(bytes.Buffer)
	previousMessages := make([]struct {
		Role    string `json:"role"`
		Content string `json:"content"`
		Command string `json:"command,omitempty"`
		Mode    string `json:"mode,omitempty"`
	}, 0)
	for i, message := range completion.Messages {
		content := message.GetString("content")
		for k, v := range mapC {
			content = strings.ReplaceAll(content, k, b+v+b)
		}
		mapCc := env.GetStringMapString("qodo.mapC")
		for k, v := range mapCc {
			content = strings.ReplaceAll(content, k, b+v+b)
		}

		if i >= len(previousMessages)-1 {
			chatInput = content
			break
		}
		previousMessages = append(previousMessages, struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			Command string `json:"command,omitempty"`
			Mode    string `json:"mode,omitempty"`
		}{
			Role:    elseOf(message.Is("role", "user"), "user", "assistant"),
			Mode:    elseOf(message.Is("role", "user"), "freeChat", ""),
			Command: elseOf(message.Is("role", "user"), "chat", ""),
			Content: content,
		})
		//role, end := response.ConvertRole(ctx, message.GetString("role"))
		//contentBuffer.WriteString(role)
		//contentBuffer.WriteString(message.GetString("content"))
		//contentBuffer.WriteString(end)
	}

	//msg := contentBuffer.String()
	//for k, v := range mapC {
	//	msg = strings.ReplaceAll(msg, k, b+v+b)
	//}
	//mapCc := env.GetStringMapString("qodo.mapC")
	//for k, v := range mapCc {
	//	msg = strings.ReplaceAll(msg, k, b+v+b)
	//}

	request = qodoRequest{
		MaxRemoteContext:  0,
		RemoteContextTags: make([]interface{}, 0),
		MaxRepoContext:    5,
		UserData: struct {
			InstallationId              string `json:"installation_id"`
			InstallationFingerprintUuid string `json:"installation_fingerprint_uuid"`
			EditorVersion               string `json:"editor_version"`
			ExtensionVersion            string `json:"extension_version"`
			OsPlatform                  string `json:"os_platform"`
			OsVersion                   string `json:"os_version"`
			EditorType                  string `json:"editor_type"`
		}{
			InstallationId:              uuid.NewString(),
			InstallationFingerprintUuid: uuid.NewString(),
			EditorVersion:               "1.97.2",
			ExtensionVersion:            "0.16.2",
			OsPlatform:                  "darwin",
			OsVersion:                   "v20.18.1",
			EditorType:                  "vscode",
		},
		Task:              "",
		ChatInput:         chatInput,
		PreviousMessages:  previousMessages,
		UserContext:       make([]interface{}, 0),
		RepoContext:       make([]interface{}, 0),
		CustomModel:       completion.Model[5:],
		SupportsArtifacts: true,
	}
	return
}

func genToken(ctx *gin.Context, env *env.Environment) (token string, err error) {
	cookies := ctx.GetString("token")
	cacheManager := cache.QodoCacheManager()
	token, err = cacheManager.GetValue(cookies)
	if token != "" || err != nil {
		return
	}

	split := strings.Split(cookies, "|")
	if len(split) < 2 {
		err = fmt.Errorf("invalid cookie format")
		return
	}

	r, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(ctx.GetString("proxies")).
		GET("https://accounts.google.com/o/oauth2/auth").
		Query("client_id", split[0]+".apps.googleusercontent.com").
		Query("scope", url.QueryEscape("email profile openid")).
		Query("response_type", "code").
		Query("access_type", "offline").
		Query("state", base64.URLEncoding.EncodeToString([]byte("windowId=1;vscode"))).
		Query("redirect_uri", "https://api.qodo.ai/v1/auth/google-login/Codium.codium").
		Header("accept", "text/html,*/*").
		Header("Accept-Language", "en-US,en;q=0.9").
		Header("Origin", "https://app.qodo.ai").
		Header("Referer", "https://app.qodo.ai/").
		Header("user-agent", userAgent).
		Header("cookie", split[1]).
		DoC(emit.Status(http.StatusFound), emit.IsHTML)
	if err != nil {
		return
	}

	location := r.Header.Get("Location")
	_ = r.Body.Close()
	u, err := url.Parse(location)
	if err != nil {
		return
	}
	query := u.Query()

	r, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(ctx.GetString("proxies")).
		GET("https://api.qodo.ai/v1/auth/google-login/Codium.codium").
		Query("state", query.Get("state")).
		Query("code", query.Get("code")).
		Query("scope", url.QueryEscape(query.Get("scope"))).
		Query("authuser", query.Get("authuser")).
		Query("prompt", query.Get("prompt")).
		Header("accept", "text/html,*/*").
		Header("Accept-Language", "en-US,en;q=0.9").
		Header("Origin", "https://app.qodo.ai").
		Header("Referer", "https://app.qodo.ai/").
		Header("user-agent", userAgent).
		Header("cookie", split[1]).
		DoS(http.StatusTemporaryRedirect)
	if err != nil {
		return
	}

	location = r.Header.Get("Location")
	_ = r.Body.Close()
	u, err = url.Parse(location)
	if err != nil {
		return
	}
	query = u.Query()
	qodoToken := query.Get("token")
	_ = r.Body.Close()

	r, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx.Request.Context()).
		Proxies(ctx.GetString("proxies")).
		POST("https://identitytoolkit.googleapis.com/v1/accounts:signInWithIdp").
		Query("key", env.GetString("qodo.key")).
		Header("Accept-Language", "en-US,en;q=0.9").
		Header("Origin", "https://app.qodo.ai").
		Header("Referer", "https://app.qodo.ai/").
		Header("user-agent", "node").
		JSONHeader().
		Body(map[string]interface{}{
			"requestUri":        "http://localhost",
			"returnSecureToken": true,
			"postBody":          "&id_token=" + qodoToken + "&providerId=google.com",
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}
	defer r.Body.Close()
	obj, err := emit.ToMap(r)
	if err != nil {
		return
	}

	accessToken, ok := obj["idToken"]
	if !ok {
		err = fmt.Errorf("grant access_token failed")
		return
	}

	token = accessToken.(string)
	_ = cacheManager.SetWithExpiration(cookies, token, time.Hour)
	return
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}
