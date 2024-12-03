package response

import "github.com/iocgo/sdk/env"

// @Inject(singleton = "false")
func New(env *env.Environment) *ContentHolder {
	return &ContentHolder{
		env,
	}
}
