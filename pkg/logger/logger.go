package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger      *zap.Logger
	sugar       *zap.SugaredLogger
	initialized bool
)

// Config 日志配置
type Config struct {
	Level      string `json:"level"`       // debug, info, warn, error
	Filename   string `json:"filename"`    // 日志文件路径
	MaxSize    int    `json:"max_size"`    // 单个文件最大大小（MB）
	MaxBackups int    `json:"max_backups"` // 保留旧文件的最大数量
	MaxAge     int    `json:"max_age"`     // 保留旧文件的最大天数
	Compress   bool   `json:"compress"`    // 是否压缩旧文件
	IsDev      bool   `json:"is_dev"`      // 是否为开发环境
}

// Init 初始化日志系统
func Init(cfg Config) error {
	// 确保日志目录存在
	logDir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 设置日志级别
	var level zapcore.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 创建编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   customCallerEncoder,
	}

	// 配置日志输出
	var core zapcore.Core
	if cfg.IsDev {
		// 开发环境：使用控制台格式输出到控制台和文件
		devEncoderConfig := encoderConfig
		devEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 添加颜色
		devEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		}
		devEncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			path := getRelativePath(caller.File)
			funcName := caller.Function
			if idx := strings.LastIndex(funcName, "."); idx > 0 {
				funcName = funcName[idx+1:]
			}
			enc.AppendString(fmt.Sprintf("%s:%d [%s]", path, caller.Line, funcName))
		}

		// 开发环境使用控制台格式
		consoleEncoder := zapcore.NewConsoleEncoder(devEncoderConfig)
		fileEncoder := zapcore.NewConsoleEncoder(devEncoderConfig)

		// 文件输出
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})

		// 控制台输出
		consoleWriter := zapcore.AddSync(os.Stdout)

		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, fileWriter, level),
			zapcore.NewCore(consoleEncoder, consoleWriter, level),
		)
	} else {
		// 生产环境：使用JSON格式输出到文件
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})

		core = zapcore.NewCore(fileEncoder, fileWriter, level)
	}

	// 创建logger实例
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
	initialized = true

	return nil
}

// ensureInitialized 确保日志系统已初始化，如果没有则自动初始化
func ensureInitialized() {
	if !initialized {
		// 使用默认配置自动初始化
		defaultConfig := Config{
			Level:      "info",
			Filename:   "./logs/app.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
			IsDev:      true, // 默认开发环境，输出到控制台
		}

		if err := Init(defaultConfig); err != nil {
			// 如果自动初始化失败，创建一个基础的logger避免panic
			encoderConfig := zapcore.EncoderConfig{
				TimeKey:    "ts",
				LevelKey:   "level",
				MessageKey: "msg",
				EncodeTime: customTimeEncoder,
			}

			core := zapcore.NewCore(
				zapcore.NewConsoleEncoder(encoderConfig),
				zapcore.AddSync(os.Stdout),
				zapcore.InfoLevel,
			)

			logger = zap.New(core)
			sugar = logger.Sugar()
			initialized = true
		}
	}
}

// 自定义时间编码器
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// 自定义调用者编码器
func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// 转换为相对路径
	path := getRelativePath(caller.File)

	// 获取函数名
	funcName := caller.Function
	if idx := strings.LastIndex(funcName, "."); idx > 0 {
		funcName = funcName[idx+1:]
	}

	// 格式化为"相对路径:行号 [函数名]"
	enc.AppendString(fmt.Sprintf("%s:%d [%s]", path, caller.Line, funcName))
}

// 获取相对路径
func getRelativePath(path string) string {
	// 获取项目根目录
	_, file, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(file)
		projectRoot := filepath.Dir(filepath.Dir(dir))
		if strings.Contains(path, projectRoot) {
			rel, err := filepath.Rel(projectRoot, path)
			if err == nil {
				return rel
			}
		}
	}
	dir, file := filepath.Split(path)
	parent := filepath.Base(dir)
	return filepath.Join(parent, file)
}

// Debug 输出Debug级别日志
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	ensureInitialized()
	logger.Debug(msg, fields...)
}

// Debugf 格式化输出Debug级别日志
func Debugf(ctx context.Context, format string, args ...interface{}) {
	ensureInitialized()
	sugar.Debugf(format, args...)
}

// Info 输出Info级别日志
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	ensureInitialized()
	logger.Info(msg, fields...)
}

// Infof 格式化输出Info级别日志
func Infof(ctx context.Context, format string, args ...interface{}) {
	ensureInitialized()
	sugar.Infof(format, args...)
}

// Warn 输出Warn级别日志
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	ensureInitialized()
	logger.Warn(msg, fields...)
}

// Warnf 格式化输出Warn级别日志
func Warnf(ctx context.Context, format string, args ...interface{}) {
	ensureInitialized()
	sugar.Warnf(format, args...)
}

// Error 输出Error级别日志
func Error(ctx context.Context, msg string, args ...interface{}) {
	ensureInitialized()
	sugar.Errorf(msg, args...)
}

// Errorf 格式化输出Error级别日志
func Errorf(ctx context.Context, format string, args ...interface{}) {
	ensureInitialized()
	sugar.Errorf(format, args...)
}

// Fatal 输出Fatal级别日志并退出程序
func Fatal(ctx context.Context, msg string, args ...interface{}) {
	ensureInitialized()
	sugar.Fatalf(msg, args...)
}

// Fatalf 格式化输出Fatal级别日志并退出程序
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	ensureInitialized()
	sugar.Fatalf(format, args...)
}

// Sync 同步日志
func Sync() error {
	if initialized {
		return logger.Sync()
	}
	return nil
}
