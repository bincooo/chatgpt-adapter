package vars

import _ "embed"

var (
	//go:embed js/dist/index.js
	Script string
)
