package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// 全局日志实例
	logger      *zap.Logger
	atomicLevel zap.AtomicLevel
	once        sync.Once
)

// 初始化日志
func init() {
	once.Do(func() {
		logger = newLogger()
	})
}

// newLogger 创建一个新的日志实例
func newLogger() *zap.Logger {
	// 创建基础的encoder配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建AtomicLevel
	atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)

	// 创建Core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		atomicLevel,
	)

	// 创建Logger
	return zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}

// Info 记录INFO级别的日志
func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// Debug 记录DEBUG级别的日志
func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

// Warn 记录WARN级别的日志
func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

// Error 记录ERROR级别的日志
func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

// Fatal 记录FATAL级别的日志，然后退出程序
func Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

// With 返回带有指定字段的Logger
func With(fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}

// SetLevel 设置日志级别
func SetLevel(level string) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}
	atomicLevel.SetLevel(zapLevel)
}
