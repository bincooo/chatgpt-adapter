package common

import (
	"chatgpt-adapter/core/logger"
	"github.com/iocgo/sdk/env"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

var cmd *exec.Cmd

func Exec(port, proxies string, stdout io.Writer, stderr io.Writer) {
	app := appPath()

	if !fileExists(app) {
		logger.Fatalf("executable file not exists: %s", app)
		return
	}

	args := []string{app, "--port", port}
	if proxies != "" {
		args = append(args, "--proxies", proxies)
	}

	cmd = exec.Command(app, args...)
	if stdout == nil {
		stdout = os.Stdout
	}
	cmd.Stdout = stdout

	if stderr == nil {
		stderr = os.Stderr
	}
	cmd.Stderr = stderr

	go func() {
		if err := cmd.Run(); err != nil {
			logger.Fatalf("executable file error: %v", err)
			return
		}
	}()

	time.Sleep(5 * time.Second)
	logger.Info("helper exec running ...")
}

func appPath() string {
	app := "bin/"
	switch runtime.GOOS {
	case "linux":
		// 可惜了，arm过不了验证
		if runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" {
			app += "linux/helper-arm64"
		} else {
			app += "linux/helper"
		}
	case "darwin":
		app += "osx/helper"
	case "windows":
		app += "windows/helper.exe"
	default:
		logger.Fatalf("Unsupported platform: %s", runtime.GOOS)
	}
	return app
}

func Exit(_ *env.Environment) {
	if cmd == nil {
		return
	}
	_ = cmd.Process.Kill()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
