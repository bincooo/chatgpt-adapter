package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

var (
	_logger *zap.Logger
	_sugar  *zap.SugaredLogger
)

type Level = zapcore.Level

var (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	FatalLevel = zapcore.FatalLevel
)

func Initialized(path string, logLevel Level) {
	if len(path) == 0 {
		path = "log"
	}
	writeSyncer := getWriter(filepath.Join(path, "/adapter.log"), 1, 3, 28)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, writeSyncer, logLevel)
	_logger = zap.New(core, zap.AddCaller())
}

func getWriter(filename string, maxsize, maxBackup, maxAge int) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxsize,
		MaxAge:     maxAge,
		MaxBackups: maxBackup,
		Compress:   false,
	}
	return zapcore.NewMultiWriteSyncer(zapcore.AddSync(lumberJackLogger), zapcore.AddSync(os.Stdout))
}

func sugar() *zap.SugaredLogger {
	if _sugar == nil {
		_sugar = _logger.Sugar()
	}
	return _sugar
}

func Logger() *zap.SugaredLogger {
	return sugar()
}
