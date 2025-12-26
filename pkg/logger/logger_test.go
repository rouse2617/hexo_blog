package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInit(t *testing.T) {
	// 测试 JSON 格式初始化
	err := Init(Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}

	// 验证 logger 已创建
	if globalLogger == nil {
		t.Error("globalLogger 未初始化")
	}
	if globalSugar == nil {
		t.Error("globalSugar 未初始化")
	}
}

func TestInitTextFormat(t *testing.T) {
	// 测试 text 格式初始化
	err := Init(Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}
}

func TestInitFileOutput(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// 测试文件输出
	err = Init(Config{
		Level:  "info",
		Format: "json",
		Output: tmpFile.Name(),
	})
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}

	// 写入日志
	Info("test message", zap.String("key", "value"))
	Sync()

	// 验证文件内容
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	if !strings.Contains(string(content), "test message") {
		t.Error("日志文件中未找到测试消息")
	}
	if !strings.Contains(string(content), "key") {
		t.Error("日志文件中未找到字段 key")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"ERROR", zapcore.ErrorLevel},
		{"unknown", zapcore.InfoLevel}, // 默认 info
		{"", zapcore.InfoLevel},        // 空字符串默认 info
	}

	for _, tt := range tests {
		result := parseLevel(tt.input)
		if result != tt.expected {
			t.Errorf("parseLevel(%q) = %v, 期望 %v", tt.input, result, tt.expected)
		}
	}
}

func TestLogMethods(t *testing.T) {
	// 使用内存 buffer 捕获日志输出
	var buf bytes.Buffer

	// 创建自定义 logger
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	globalLogger = zap.New(core)
	globalSugar = globalLogger.Sugar()

	// 测试各种日志方法
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	Debugf("debug %s", "formatted")
	Infof("info %s", "formatted")
	Warnf("warn %s", "formatted")
	Errorf("error %s", "formatted")

	output := buf.String()

	// 验证输出包含各级别日志
	if !strings.Contains(output, "debug message") {
		t.Error("未找到 debug message")
	}
	if !strings.Contains(output, "info message") {
		t.Error("未找到 info message")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("未找到 warn message")
	}
	if !strings.Contains(output, "error message") {
		t.Error("未找到 error message")
	}
	if !strings.Contains(output, "debug formatted") {
		t.Error("未找到 debug formatted")
	}
}

func TestGetLogger(t *testing.T) {
	// 重置全局 logger
	globalLogger = nil
	globalSugar = nil

	// 获取 logger 应该返回默认 logger
	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger 返回 nil")
	}

	sugar := GetSugar()
	if sugar == nil {
		t.Error("GetSugar 返回 nil")
	}
}

func TestWith(t *testing.T) {
	err := Init(Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}

	// 测试 With 方法
	childLogger := With(zap.String("component", "test"))
	if childLogger == nil {
		t.Error("With 返回 nil")
	}

	// 测试 WithFields 方法
	childSugar := WithFields("component", "test")
	if childSugar == nil {
		t.Error("WithFields 返回 nil")
	}
}
