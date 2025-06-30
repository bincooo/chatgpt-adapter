package common

import (
	"os"
	"os/signal"
	"syscall"

	"adapter/module/env"
)

var (
	inits = make([]func(env *env.Environ), 0)
	exits = make([]func(env *env.Environ), 0)
)

func AddInitialized(apply func(env *env.Environ)) { inits = append(inits, apply) }
func AddExited(apply func(env *env.Environ))      { exits = append(exits, apply) }
func Initialized(env *env.Environ) {
	for _, apply := range inits {
		apply(env)
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func(ch chan os.Signal) {
		<-ch
		for _, apply := range exits {
			apply(env)
		}
		os.Exit(0)
	}(osSignal)
}
