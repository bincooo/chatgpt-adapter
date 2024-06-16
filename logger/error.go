package logger

import (
	"errors"
	"fmt"
	"path"
	"runtime"
	"strconv"
	"strings"
)

type StackError struct {
	err          error
	functionInfo string
}

func (e StackError) Error() string {
	err := e.rootErr()
	return fmt.Sprintf("===== STACK ERROR %s =====\n%v", err.functionInfo, err.err)
}

func (e StackError) rootErr() StackError {
	var err = e
	var se StackError
	for {
		if errors.As(err.err, &se) {
			err = se
			continue
		}
		break
	}
	return err
}

func (e StackError) OriginError() error {
	return e.rootErr().err
}

func WarpError(err error) error {
	if err == nil {
		return nil
	}

	trimPackage := func(pkg string) string {
		if pkg == "" {
			return pkg
		}
		slice := strings.Split(pkg, "/")
		length := len(slice)
		if length <= 2 {
			return pkg
		}
		return slice[length-2] + "/" + slice[length-1]
	}

	trimL := func(prefix string) string {
		if prefix == "" {
			return prefix
		}
		if strings.HasPrefix(prefix, "/") {
			return prefix[1:]
		}
		return prefix
	}

	var functionInfo = "main"
	// 尝试获取上层栈
	pcs := make([]uintptr, 3)
	depth := runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	var frame runtime.Frame
	for f, next := frames.Next(); next; f, next = frames.Next() {
		if strings.HasSuffix(f.Function, "WarpError") {
			f, next = frames.Next()
			if next {
				frame = f
				functionInfo = frame.Function
			}
			break
		}
	}

	if functionInfo != "main" {
		slice := strings.Split(frame.File, trimPackage(path.Dir(functionInfo)))
		if len(slice) > 1 {
			functionInfo = trimL(slice[1]) + ":" + strconv.Itoa(frame.Line)
		}
	}

	return StackError{
		err: err,
		//
		functionInfo: functionInfo,
	}
}
