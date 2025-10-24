package logger

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// 创建测试用的输出缓冲区
	var buf bytes.Buffer

	// 创建测试日志器
	config := &Config{
		Level:      DEBUG,
		Output:     &buf,
		TimeFormat: "2006-01-02 15:04:05",
		ShowCaller: false,
	}

	logger := NewLogger(config)

	// 测试不同级别的日志
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	output := buf.String()

	// 验证输出包含预期的消息
	if !strings.Contains(output, "DEBUG") {
		t.Error("Debug message not found in output")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("Info message not found in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("Warning message not found in output")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("Error message not found in output")
	}
}

func TestLogLevel(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:      WARN,
		Output:     &buf,
		TimeFormat: "2006-01-02 15:04:05",
		ShowCaller: false,
	}

	logger := NewLogger(config)

	// 这些消息不应该被记录
	logger.Debug("Debug message")
	logger.Info("Info message")

	// 这些消息应该被记录
	logger.Warn("Warning message")
	logger.Error("Error message")

	output := buf.String()

	// 验证低级别消息没有被记录
	if strings.Contains(output, "DEBUG") {
		t.Error("Debug message should not be logged at WARN level")
	}
	if strings.Contains(output, "INFO") {
		t.Error("Info message should not be logged at WARN level")
	}

	// 验证高级别消息被记录
	if !strings.Contains(output, "WARN") {
		t.Error("Warning message should be logged at WARN level")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("Error message should be logged at WARN level")
	}
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:      INFO,
		Output:     &buf,
		TimeFormat: "2006-01-02 15:04:05",
		ShowCaller: false,
	}

	logger := NewLogger(config)
	ctx := context.Background()
	ctxLogger := logger.WithContext(ctx)

	ctxLogger.Info("Context message")

	output := buf.String()
	if !strings.Contains(output, "Context message") {
		t.Error("Context message not found in output")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"info", INFO},
		{"INFO", INFO},
		{"warn", WARN},
		{"WARN", WARN},
		{"warning", WARN},
		{"WARNING", WARN},
		{"error", ERROR},
		{"ERROR", ERROR},
		{"fatal", FATAL},
		{"FATAL", FATAL},
		{"unknown", INFO},
		{"", INFO},
	}

	for _, test := range tests {
		result := ParseLevel(test.input)
		if result != test.expected {
			t.Errorf("ParseLevel(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestLogConfig(t *testing.T) {
	config := &LogConfig{
		Level:      "debug",
		Output:     "stdout",
		TimeFormat: "2006-01-02 15:04:05.000",
		ShowCaller: true,
	}

	loggerConfig := config.ToLoggerConfig()

	if loggerConfig.Level != DEBUG {
		t.Error("Level not parsed correctly")
	}
	if loggerConfig.Output != os.Stdout {
		t.Error("Output not parsed correctly")
	}
	if loggerConfig.TimeFormat != "2006-01-02 15:04:05.000" {
		t.Error("TimeFormat not set correctly")
	}
	if !loggerConfig.ShowCaller {
		t.Error("ShowCaller not set correctly")
	}
}
