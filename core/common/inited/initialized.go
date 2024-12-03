package inited

import (
	"github.com/iocgo/sdk/env"
	"os"
	"os/signal"
	"syscall"
)

var (
	inits = make([]func(env *env.Environment), 0)
	exits = make([]func(env *env.Environment), 0)
)

func AddInitialized(apply func(env *env.Environment)) { inits = append(inits, apply) }
func AddExited(apply func(env *env.Environment))      { exits = append(exits, apply) }
func Initialized(env *env.Environment) {
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
