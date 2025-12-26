package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 全局 logger 实例
var (
	globalLogger *zap.Logger
	globalSugar  *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	Level  string // debug / info / warn / error
	Format string // json / text
	Output string // stdout / 文件路径
}

// Init 初始化日志
func Init(cfg Config) error {
	// 解析日志级别
	level := parseLevel(cfg.Level)

	// 创建编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 根据格式选择编码器
	var encoder zapcore.Encoder
	if strings.ToLower(cfg.Format) == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// text 格式使用 console encoder
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 配置输出
	var writeSyncer zapcore.WriteSyncer
	if strings.ToLower(cfg.Output) == "stdout" || cfg.Output == "" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		// 输出到文件
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writeSyncer = zapcore.AddSync(file)
	}

	// 创建 core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 创建 logger
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	globalSugar = globalLogger.Sugar()

	return nil
}

// parseLevel 解析日志级别
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sync 刷新日志缓冲
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// GetLogger 获取原始 zap.Logger
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// 返回默认 logger
		globalLogger, _ = zap.NewDevelopment()
		globalSugar = globalLogger.Sugar()
	}
	return globalLogger
}

// GetSugar 获取 SugaredLogger
func GetSugar() *zap.SugaredLogger {
	if globalSugar == nil {
		GetLogger()
	}
	return globalSugar
}

// Debug 输出 debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 输出 info 级别日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 输出 warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 输出 error 级别日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 输出 fatal 级别日志并退出
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Debugf 格式化输出 debug 级别日志
func Debugf(template string, args ...interface{}) {
	GetSugar().Debugf(template, args...)
}

// Infof 格式化输出 info 级别日志
func Infof(template string, args ...interface{}) {
	GetSugar().Infof(template, args...)
}

// Warnf 格式化输出 warn 级别日志
func Warnf(template string, args ...interface{}) {
	GetSugar().Warnf(template, args...)
}

// Errorf 格式化输出 error 级别日志
func Errorf(template string, args ...interface{}) {
	GetSugar().Errorf(template, args...)
}

// Fatalf 格式化输出 fatal 级别日志并退出
func Fatalf(template string, args ...interface{}) {
	GetSugar().Fatalf(template, args...)
}

// With 创建带有额外字段的 logger
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// WithFields 创建带有额外字段的 sugar logger
func WithFields(args ...interface{}) *zap.SugaredLogger {
	return GetSugar().With(args...)
}
