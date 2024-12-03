package pg

import (
	"chatgpt-adapter/core/gin/inter"
	"github.com/iocgo/sdk/env"

	_ "github.com/iocgo/sdk"
)

// @Inject(name = "pg-adapter")
func New(env *env.Environment) inter.Adapter { return &pg{env: env} }
