package qodo

import (
	"bytes"
	"chatgpt-adapter/core/cache"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
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
)

type qodoRequest struct {
	Tools            model.Keyv[interface{}] `json:"tools"`
	CustomModel      *string                 `json:"custom_model,omitempty"`
	ProjectsRootPath []string                `json:"projects_root_path"`
	UserRequest      string                  `json:"user_request"`
	UserData         struct {
		OsPlatform                  string `json:"os_platform"`
		InstallationFingerprintUuid string `json:"installation_fingerprint_uuid"`
		EditorVersion               string `json:"editor_version"`
		OsVersion                   string `json:"os_version"`
		ExtensionVersion            string `json:"extension_version"`
		InstallationId              string `json:"installation_id"`
		EditorType                  string `json:"editor_type"`
	} `json:"user_data"`
	SessionId   string        `json:"session_id"`
	UserContext []interface{} `json:"user_context"`
}

func fetch(ctx *gin.Context, proxied string, request qodoRequest) (response *http.Response, err error) {
	token, err := genToken(ctx, env.Env)
	dateStr := time.Now().Format("20060102-")
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxied).
		POST("https://api.gen.qodo.ai/v2/agentic/start-task").
		JSONHeader().
		//Header("accept", "text/plain").
		Header("host", "api.gen.qodo.ai").
		Header("user-agent", "axios/1.7.9").
		Header("accept-encoding", "gzip, compress, deflate, br").
		Header("authorization", "Bearer "+token).
		Header("Request-id", uuid.NewString()).
		Header("session-id", dateStr+uuid.NewString()).
		Body(request).
		DoS(http.StatusOK)
	return
}

func convertRequest(ctx *gin.Context, env *env.Environment, completion model.Completion) (request qodoRequest, err error) {
	contentBuffer := new(bytes.Buffer)
	for _, message := range completion.Messages {
		content := message.GetString("content")
		//content = _hook(content)

		role, end := response.ConvertRole(ctx, message.GetString("role"))
		contentBuffer.WriteString(role)
		contentBuffer.WriteString(content)
		contentBuffer.WriteString(end)
	}

	mod := completion.Model[5:]
	request = qodoRequest{
		Tools: model.Keyv[interface{}]{
			"GIT": []model.Keyv[interface{}]{
				{
					"autoApproved": true,
					"inputSchema": model.Keyv[interface{}]{
						"required": []string{
							"path",
						},
						"type": "object",
						"properties": model.Keyv[interface{}]{
							"path": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Repository path",
							},
							"remote": model.Keyv[interface{}]{
								"default":     "origin",
								"type":        "string",
								"description": "Remote name (defaults to origin)",
							},
						},
						"$schema":              "http://json-schema.org/draft-07/schema#",
						"additionalProperties": false,
					},
					"name":        "git_remote_url",
					"description": "Retrieves the URL of a git remote (defaults to 'origin')",
				},
				{
					"name":         "git_branches",
					"autoApproved": true,
					"description":  "Lists all branches in the repository",
					"inputSchema": model.Keyv[interface{}]{
						"type": "object",
						"properties": model.Keyv[interface{}]{
							"all": model.Keyv[interface{}]{
								"description": "Include remote branches",
								"default":     false,
								"type":        "boolean",
							},
							"path": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Repository path",
							},
						},
						"required": []string{
							"path",
						},
						"additionalProperties": false,
						"$schema":              "http://json-schema.org/draft-07/schema#",
					},
				},
				{
					"name": "git_changes",
					"inputSchema": model.Keyv[interface{}]{
						"required": []string{
							"path",
						},
						"properties": model.Keyv[interface{}]{
							"filepath": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Specific file or directory path to check changes",
							},
							"path": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Repository path",
							},
							"branch": model.Keyv[interface{}]{
								"default":     "HEAD",
								"description": "Branch name to compare (default to HEAD)",
								"type":        "string",
							},
						},
						"$schema":              "http://json-schema.org/draft-07/schema#",
						"type":                 "object",
						"additionalProperties": false,
					},
					"description":  "Shows current changes in the repository",
					"autoApproved": true,
				},
				{
					"name":         "git_file_history",
					"description":  "Shows commit history for a specific file",
					"autoApproved": true,
					"inputSchema": model.Keyv[interface{}]{
						"$schema": "http://json-schema.org/draft-07/schema#",
						"required": []string{
							"path",
							"filepath",
						},
						"additionalProperties": false,
						"type":                 "object",
						"properties": model.Keyv[interface{}]{
							"path": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Repository path",
							},
							"limit": model.Keyv[interface{}]{
								"description": "Number of commits to show",
								"type":        "number",
								"default":     10,
							},
							"filepath": model.Keyv[interface{}]{
								"description": "File path to get history for",
								"type":        "string",
							},
						},
					},
				},
			},
			"Code Navigation": []model.Keyv[interface{}]{
				{
					"inputSchema": model.Keyv[interface{}]{
						"properties": model.Keyv[interface{}]{
							"ast_node_line": model.Keyv[interface{}]{
								"description": "Line number of the AST node to retrieve dependencies for if there are multiple AST nodes with the same name",
								"type":        "number",
							},
							"ast_node_name": model.Keyv[interface{}]{
								"description": "Name of the AST node name to retrieve dependencies for",
								"type":        "string",
							},
							"path": model.Keyv[interface{}]{
								"description": "Full absolute path to the file where the AST node is located",
								"type":        "string",
							},
						},
						"required": []string{
							"path",
							"ast_node_name",
						},
						"type":                 "object",
						"$schema":              "http://json-schema.org/draft-07/schema#",
						"additionalProperties": false,
					},
					"description":  "Given a function or class identify all external dependencies including objects and function and return the code implementation of those dependencies",
					"name":         "get_code_dependencies",
					"autoApproved": true,
				},
				{
					"description":  "Given a AST node (function, class, var) identify usages of that component across the code base.",
					"name":         "find_code_usages",
					"autoApproved": true,
					"inputSchema": model.Keyv[interface{}]{
						"required": []string{
							"path",
							"ast_node_name",
						},
						"properties": model.Keyv[interface{}]{
							"exclude_patterns": model.Keyv[interface{}]{
								"type":        "array",
								"description": "Glob patterns to exclude (e.g., '**/node_modules/**', '**/venv/**', '**/__pycache__/**' etc.)",
								"items": model.Keyv[interface{}]{
									"type": "string",
								},
							},
							"ast_node_name": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Name of the AST node to find usages for",
							},
							"path": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Full absolute path to the file where the AST node is located",
							},
						},
						"type":                 "object",
						"additionalProperties": false,
						"$schema":              "http://json-schema.org/draft-07/schema#",
					},
				},
			},
			"Fetch URL": []model.Keyv[interface{}]{
				{
					"inputSchema": model.Keyv[interface{}]{
						"additionalProperties": false,
						"required": []string{
							"url",
						},
						"$schema": "http://json-schema.org/draft-07/schema#",
						"properties": model.Keyv[interface{}]{
							"url": model.Keyv[interface{}]{
								"description": "URL of the website to fetch",
								"type":        "string",
							},
							"headers": model.Keyv[interface{}]{
								"additionalProperties": model.Keyv[interface{}]{
									"type": "string",
								},
								"type":        "object",
								"description": "Optional headers to include in the request",
							},
						},
						"type": "object",
					},
					"name":        "fetch_html",
					"description": "Fetch a website and return the content as HTML",
				},
				{
					"inputSchema": model.Keyv[interface{}]{
						"$schema": "http://json-schema.org/draft-07/schema#",
						"required": []string{
							"url",
						},
						"type":                 "object",
						"additionalProperties": false,
						"properties": model.Keyv[interface{}]{
							"headers": model.Keyv[interface{}]{
								"description": "Optional headers to include in the request",
								"type":        "object",
								"additionalProperties": model.Keyv[interface{}]{
									"type": "string",
								},
							},
							"url": model.Keyv[interface{}]{
								"type":        "string",
								"description": "URL of the website to fetch",
							},
						},
					},
					"description": "Fetch a website and return the content as Markdown",
					"name":        "fetch_markdown",
				},
			},
			"File System": []model.Keyv[interface{}]{
				{
					"autoApproved": true,
					"inputSchema": model.Keyv[interface{}]{
						"required": []string{
							"path",
						},
						"type":                 "object",
						"$schema":              "http://json-schema.org/draft-07/schema#",
						"additionalProperties": false,
						"properties": model.Keyv[interface{}]{
							"path": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Directory path to analyze",
							},
							"options": model.Keyv[interface{}]{
								"additionalProperties": false,
								"type":                 "object",
								"properties": model.Keyv[interface{}]{
									"maxDepth": model.Keyv[interface{}]{
										"type":        "number",
										"description": "Maximum depth to traverse",
									},
									"includeDotFiles": model.Keyv[interface{}]{
										"description": "Whether to include hidden files",
										"type":        "boolean",
									},
									"onlyDirs": model.Keyv[interface{}]{
										"description": "Only show directories",
										"type":        "boolean",
									},
									"ignore": model.Keyv[interface{}]{
										"items": model.Keyv[interface{}]{
											"type": "string",
										},
										"description": "Patterns to ignore",
										"type":        "array",
									},
								},
							},
						},
					},
					"name":        "get_directory_tree",
					"description": "Generates a hierarchical tree visualization of a directory structure. Supports filtering of common development artifacts (node_modules, .git, etc.), max depth limits, and dot-file inclusion/exclusion. Output uses standard tree formatting with branch indicators.",
				},
				{
					"inputSchema": model.Keyv[interface{}]{
						"type": "object",
						"properties": model.Keyv[interface{}]{
							"paths": model.Keyv[interface{}]{
								"type":        "array",
								"description": "Full absolute path to the files to read",
								"items": model.Keyv[interface{}]{
									"type": "string",
								},
							},
						},
						"required": []string{
							"paths",
						},
						"additionalProperties": false,
						"$schema":              "http://json-schema.org/draft-07/schema#",
					},
					"autoApproved": true,
					"description":  "Read the contents of one or more files simultaneously. Each file's content is returned with its path as a reference. Failed reads for individual files won't stop the entire operation. Only works within allowed directories.",
					"name":         "read_files",
				},
				{
					"description": "Recursively search for files and directories matching a pattern. Searches through all subdirectories from the starting path. The search is case-insensitive and matches partial names. Returns full paths to all matching items. Great for finding files when you don't know their exact location. Only searches within allowed directories.",
					"name":        "search_for_files",
					"inputSchema": model.Keyv[interface{}]{
						"properties": model.Keyv[interface{}]{
							"path": model.Keyv[interface{}]{
								"description": "Full absolute path to start the search. Must be within the projects directories",
								"type":        "string",
							},
							"pattern": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Pattern to search for in file names",
							},
						},
						"additionalProperties": false,
						"required": []string{
							"path",
							"pattern",
						},
						"type":    "object",
						"$schema": "http://json-schema.org/draft-07/schema#",
					},
					"autoApproved": true,
				},
				{
					"description":  "Performs fast recursive file content searches within a workspace directory. Supports both plain text and regex patterns, with configurable file filtering via glob patterns. Can exclude specific paths (like node_modules, dist, etc.) and returns matches with file paths and line numbers.",
					"name":         "search_in_files",
					"autoApproved": true,
					"inputSchema": model.Keyv[interface{}]{
						"properties": model.Keyv[interface{}]{
							"filePattern": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Glob pattern to filter files (e.g., '**/*.ts')",
							},
							"pattern": model.Keyv[interface{}]{
								"type":        "string",
								"description": "String or regex pattern to search for in files",
								"minLength":   1,
							},
							"path": model.Keyv[interface{}]{
								"type":        "string",
								"description": "Full absolute path to start the search. Must be within the projects directories",
							},
							"excludePatterns": model.Keyv[interface{}]{
								"type":        "array",
								"description": "Glob patterns to exclude (e.g., '**/node_modules/**', '**/venv/**', '**/__pycache__/**' etc.)",
								"items": model.Keyv[interface{}]{
									"type": "string",
								},
							},
						},
						"additionalProperties": false,
						"$schema":              "http://json-schema.org/draft-07/schema#",
						"type":                 "object",
						"required": []string{
							"path",
							"pattern",
						},
					},
				},
			},
		},
		CustomModel: elseOf(mod == "claude-3-7-sonnet", nil, &mod),
		ProjectsRootPath: []string{
			"/Users/alice/code-workspace",
		},
		UserRequest: contentBuffer.String(),
		UserData: struct {
			OsPlatform                  string `json:"os_platform"`
			InstallationFingerprintUuid string `json:"installation_fingerprint_uuid"`
			EditorVersion               string `json:"editor_version"`
			OsVersion                   string `json:"os_version"`
			ExtensionVersion            string `json:"extension_version"`
			InstallationId              string `json:"installation_id"`
			EditorType                  string `json:"editor_type"`
		}{
			OsPlatform:                  "darwin",
			InstallationFingerprintUuid: uuid.NewString(),
			EditorVersion:               "1.98.2",
			OsVersion:                   "v20.18.2",
			ExtensionVersion:            "1.0.4",
			InstallationId:              uuid.NewString(),
			EditorType:                  "vscode",
		},
		SessionId:   uuid.NewString(),
		UserContext: make([]interface{}, 0),
	}
	return
}

//// 毁灭吧，赶紧的
//func _hook(target string) string {
//	textQuoted := strconv.QuoteToASCII(target)
//	return textQuoted[1 : len(textQuoted)-1]
//}

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
