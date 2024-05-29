package logger

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	project    = ""
	projectDir = ""
)

func InitLogger(basePath string, level logrus.Level) {
	logrus.SetLevel(level)
	if len(basePath) == 0 {
		basePath = "log"
	}

	writer, err := rotatelogs.New(
		filepath.Join(basePath, "background-%Y-%m-%d.log"),
		//日志最大保存时间
		rotatelogs.WithMaxAge(7*24*time.Hour),
		////设置日志切割时间间隔(1天)(隔多久分割一次)
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		Fatal(err)
	}

	writers := []io.Writer{writer, os.Stdout}
	logrus.SetOutput(io.MultiWriter(writers...))
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:              true,
		TimestampFormat:       "2006-01-02 15:04:05",
		CallerFirst:           true,
		NoColors:              true,
		CustomCallerFormatter: CustomCallerFormatter,
	})
	logrus.SetReportCaller(true)
}

func CustomCallerFormatter(frame *runtime.Frame) string {
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

	trimProject := func(file string) string {
		if !strings.HasPrefix(file, project+"/") {
			return file
		}
		return file[len(project)+1:]
	}

	// 尝试获取上层栈
	pcs := make([]uintptr, 10)
	depth := runtime.Callers(10, pcs)
	frames := runtime.CallersFrames(pcs[:depth])
	for f, next := frames.Next(); next; f, next = frames.Next() {
		if f.PC == frame.PC {
			if f, next = frames.Next(); next {
				frame = &f
				break
			}
		}
	}

	main := strings.HasPrefix(frame.Function, "main.")
	slice := strings.Split(frame.File, trimPackage(path.Dir(frame.Function)))
	if !main && len(slice) > 1 {
		return " <" + trimProject(path.Dir(frame.Function)) + "> " + trimL(slice[1]) + ":" + strconv.Itoa(frame.Line) + " |"
	}

	root := path.Dir(frame.Function)
	if main {
		root = "main"
	}

	file := frame.File
	if strings.HasPrefix(file, projectDir+"/") {
		file = file[len(projectDir)+1:]
	}

	return " <" + trimProject(root) + "> " + file + ":" + strconv.Itoa(frame.Line) + " |"
}

func Trace(args ...interface{}) {
	logrus.Trace(args...)
}

func Tracef(format string, args ...interface{}) {
	logrus.Tracef(format, args...)
}

func Debug(args ...interface{}) {
	logrus.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func Info(args ...interface{}) {
	logrus.Info(args...)
}

func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

func Warn(args ...interface{}) {
	logrus.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func Error(args ...interface{}) {
	logrus.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	logrus.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}
