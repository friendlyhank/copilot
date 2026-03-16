package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// Level 日志级别
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// F 创建字段快捷方式
func F(key string, value interface{}) Field {
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

// stdLogger 标准日志实现
type stdLogger struct {
	mu     sync.Mutex
	level  Level
	out    io.Writer
	prefix string
	fields []Field
}

var (
	defaultLogger *stdLogger
	once          sync.Once
)

// New 创建日志实例
func New(level, output, format string) Logger {
	var l Level
	switch level {
	case "debug":
		l = LevelDebug
	case "info":
		l = LevelInfo
	case "warn":
		l = LevelWarn
	case "error":
		l = LevelError
	default:
		l = LevelInfo
	}

	var out io.Writer
	switch output {
	case "stdout":
		out = os.Stdout
	case "stderr", "":
		out = os.Stderr
	default:
		f, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			out = os.Stderr
		} else {
			out = f
		}
	}

	return &stdLogger{
		level: l,
		out:   out,
	}
}

// Default 获取默认日志实例
func Default() Logger {
	once.Do(func() {
		defaultLogger = &stdLogger{
			level: LevelInfo,
			out:   os.Stderr,
		}
	})
	return defaultLogger
}

// SetDefault 设置默认日志实例
func SetDefault(l Logger) {
	if sl, ok := l.(*stdLogger); ok {
		defaultLogger = sl
	}
}

func (l *stdLogger) log(level Level, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	allFields := append(l.fields, fields...)
	
	fieldStr := ""
	for _, f := range allFields {
		fieldStr += fmt.Sprintf(" %s=%v", f.Key, f.Value)
	}

	prefix := l.prefix
	if prefix != "" {
		prefix = "[" + prefix + "] "
	}

	log.New(l.out, "", log.LstdFlags).Printf("[%s] %s%s%s\n", level, prefix, msg, fieldStr)
}

func (l *stdLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

func (l *stdLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

func (l *stdLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

func (l *stdLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

func (l *stdLogger) With(fields ...Field) Logger {
	return &stdLogger{
		level:  l.level,
		out:    l.out,
		prefix: l.prefix,
		fields: append(l.fields, fields...),
	}
}

func (l *stdLogger) WithPrefix(prefix string) Logger {
	return &stdLogger{
		level:  l.level,
		out:    l.out,
		prefix: prefix,
		fields: l.fields,
	}
}

// 全局便捷函数
func Debug(msg string, fields ...Field) { Default().Debug(msg, fields...) }
func Info(msg string, fields ...Field)  { Default().Info(msg, fields...) }
func Warn(msg string, fields ...Field)  { Default().Warn(msg, fields...) }
func Error(msg string, fields ...Field) { Default().Error(msg, fields...) }
func With(fields ...Field) Logger       { return Default().With(fields...) }
func WithPrefix(prefix string) Logger   { return Default().WithPrefix(prefix) }
