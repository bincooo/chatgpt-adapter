package hf

import (
	"chatgpt-adapter/core/gin/inter"
	"github.com/iocgo/sdk/env"

	_ "github.com/iocgo/sdk"
)

// @Inject(name = "hf-adapter")
func New(env *env.Environment) inter.Adapter { return &api{env: env} }
