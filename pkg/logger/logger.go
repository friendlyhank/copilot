package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

// F 创建字段快捷方式
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	With(fields ...Field) Logger
	WithPrefix(prefix string) Logger
}

// slogLogger slog 实现
type slogLogger struct {
	logger *slog.Logger
}

var (
	defaultLogger Logger
	initOnce      sync.Once
)

// Config 日志配置
type Config struct {
	Level      string // debug, info, warn, error
	Output     string // stdout, stderr, 或文件路径
	Format     string // json, text
	MaxSize    int    // MB, 日志文件最大大小
	MaxBackups int    // 保留的旧文件数量
	MaxAge     int    // 保留天数
	Compress   bool   // 是否压缩旧文件
}

// New 创建日志实例
func New(cfg Config) Logger {
	// 解析日志级别
	level := parseLevel(cfg.Level)

	// 创建输出 writer
	writer := createWriter(cfg)

	// 创建 handler
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	return &slogLogger{
		logger: slog.New(handler),
	}
}

// parseLevel 解析日志级别
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// createWriter 创建输出 writer
func createWriter(cfg Config) io.Writer {
	var writer io.Writer

	switch cfg.Output {
	case "stdout":
		writer = os.Stdout
	case "stderr", "":
		writer = os.Stderr
	default:
		// 文件输出，确保目录存在
		dir := filepath.Dir(cfg.Output)
		if dir != "" && dir != "." {
			os.MkdirAll(dir, 0755)
		}

		// 日志轮转参数
		maxSize := cfg.MaxSize
		if maxSize <= 0 {
			maxSize = 100 // 默认 100MB
		}
		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 3 // 默认保留 3 个备份
		}
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 30 // 默认保留 30 天
		}

		writer = &lumberjack.Logger{
			Filename:   cfg.Output,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   cfg.Compress,
		}
	}

	return writer
}

// Default 获取默认日志实例
func Default() Logger {
	initOnce.Do(func() {
		if defaultLogger == nil {
			defaultLogger = New(Config{
				Level:  "info",
				Output: "stderr",
				Format: "text",
			})
		}
	})
	return defaultLogger
}

// SetDefault 设置默认日志实例
func SetDefault(l Logger) {
	defaultLogger = l
	// 确保 once 执行过，避免后续 Default() 覆盖
	initOnce.Do(func() {})
}

// fieldsToAttrs 将 Field 转换为 slog.Attr
func fieldsToAttrs(fields ...Field) []any {
	attrs := make([]any, len(fields)*2)
	for i, f := range fields {
		attrs[i*2] = f.Key
		attrs[i*2+1] = f.Value
	}
	return attrs
}

func (l *slogLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fieldsToAttrs(fields...)...)
}

func (l *slogLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fieldsToAttrs(fields...)...)
}

func (l *slogLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fieldsToAttrs(fields...)...)
}

func (l *slogLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToAttrs(fields...)...)
}

func (l *slogLogger) With(fields ...Field) Logger {
	attrs := fieldsToAttrs(fields...)
	return &slogLogger{
		logger: l.logger.With(attrs...),
	}
}

func (l *slogLogger) WithPrefix(prefix string) Logger {
	return &slogLogger{
		logger: l.logger.With("component", prefix),
	}
}

// 全局便捷函数
func Debug(msg string, fields ...Field) { Default().Debug(msg, fields...) }
func Info(msg string, fields ...Field)  { Default().Info(msg, fields...) }
func Warn(msg string, fields ...Field)  { Default().Warn(msg, fields...) }
func Error(msg string, fields ...Field) { Default().Error(msg, fields...) }
func With(fields ...Field) Logger       { return Default().With(fields...) }
func WithPrefix(prefix string) Logger   { return Default().WithPrefix(prefix) }