package logger

import (
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// 全局日志实例
	logger      *zap.Logger
	atomicLevel zap.AtomicLevel
	once        sync.Once
	currentConfig *LogConfig
)

// LogConfig 日志配置结构
type LogConfig struct {
	Level     string
	Format    string
	Output    string
	MaskSensitive bool
}

// 初始化日志
func init() {
	once.Do(func() {
		// 使用默认配置初始化
		currentConfig = &LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
			MaskSensitive: true,
		}
		logger = newLogger(currentConfig)
	})
}

// newLogger 创建一个新的日志实例
func newLogger(config *LogConfig) *zap.Logger {
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
	var zapLevel zapcore.Level
	switch config.Level {
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
	atomicLevel = zap.NewAtomicLevelAt(zapLevel)

	// 根据配置创建输出
	var writeSyncer zapcore.WriteSyncer
	if config.Output == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else if config.Output == "stderr" {
		writeSyncer = zapcore.AddSync(os.Stderr)
	} else {
		// 文件输出
		writeSyncer = createFileWriter(config.Output)
	}

	// 根据格式选择编码器
	var encoder zapcore.Encoder
	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// text格式
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 创建Core
	core := zapcore.NewCore(
		encoder,
		writeSyncer,
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

// createFileWriter 创建文件写入器
func createFileWriter(filePath string) zapcore.WriteSyncer {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// 如果创建目录失败，回退到标准输出
		return zapcore.AddSync(os.Stdout)
	}

	// 打开文件，如果不存在则创建，追加写入
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// 如果打开文件失败，回退到标准输出
		return zapcore.AddSync(os.Stdout)
	}

	return zapcore.AddSync(file)
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

// UpdateConfig 更新日志配置
func UpdateConfig(level, format, output string, maskSensitive bool) {
	currentConfig = &LogConfig{
		Level:  level,
		Format: format,
		Output: output,
		MaskSensitive: maskSensitive,
	}
	
	// 重新创建日志实例
	newLogger := newLogger(currentConfig)
	
	// 替换全局日志实例
	logger = newLogger
}

// GetLogFilePath 获取日志文件路径
func GetLogFilePath() string {
	if currentConfig != nil && currentConfig.Output != "stdout" && currentConfig.Output != "stderr" {
		return currentConfig.Output
	}
	return ""
}
