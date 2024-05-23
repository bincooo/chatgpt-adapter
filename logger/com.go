package logger

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/lestrrat-go/file-rotatelogs"
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

var projectDir string

func Init(basePath string, level logrus.Level) {
	dir, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	projectDir = dir

	logrus.SetLevel(level)
	if len(basePath) == 0 {
		basePath = "log"
	}

	writer, err := rotatelogs.New(
		filepath.Join(basePath, "%Y-%m-%d.log"),
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
	trimPKG := func(pkg string) string {
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

	trimLS := func(file string) string {
		if file == "" {
			return file
		}
		if strings.HasPrefix(file, "/") {
			return file[1:]
		}
		return file
	}

	slice := strings.Split(frame.File, trimPKG(path.Dir(frame.Function)))
	if len(slice) > 1 {
		return " <" + path.Dir(frame.Function) + "> " + trimLS(slice[1]) + ":" + strconv.Itoa(frame.Line) + " |"
	}

	file := strings.TrimLeft(frame.File, projectDir)
	return " <" + path.Dir(frame.Function) + "> " + file + ":" + strconv.Itoa(frame.Line) + " |"
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
