package common

import (
	"fmt"
	"strings"
)

var builtinTypesLower = map[string]string{
	".css":  "text/css; charset=utf-8",
	".gif":  "image/gif",
	".htm":  "text/html; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".js":   "text/javascript; charset=utf-8",
	".mjs":  "text/javascript; charset=utf-8",
	".pdf":  "application/pdf",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".wasm": "application/wasm",
	".webp": "image/webp",
	".xml":  "text/xml; charset=utf-8",
}

func MimeToSuffix(mime string) (string, error) {
	for k, v := range builtinTypesLower {
		if strings.Contains(v, mime) {
			return k, nil
		}
	}
	return "", fmt.Errorf("mime type `%s` not supported", mime)
}
