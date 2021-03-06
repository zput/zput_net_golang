// Package log is from https://github.com/micro/go-micro/blob/master/util/log/log.go
package log

import (
	"fmt"
	stdlog "log"
	"os"
)

// Logger is a generic logging interface
type Logger interface {
	Log(v ...interface{})
	Logf(format string, v ...interface{})
}

// Level is a log level
type Level int

const (
	// LevelFatal fatal level
	LevelFatal Level = iota + 1
	// LevelError error level
	LevelError
	// LevelWarn info level
	LevelWarn
	// LevelInfo info level
	LevelInfo
	// LevelDebug debug level
	LevelDebug
)

func (this *Level)String()string{
	switch *this {
	case LevelFatal:
		return "[F]"
	case LevelError:
		return "[E]"
	case LevelWarn:
		return "[W]"
	case LevelInfo:
		return "[I]"
	case LevelDebug:
		return "[D]"
	}
	return ""
}

var (
	// the local logger
	logger Logger = &defaultLogLogger{}

	// default log level is info
	//level = LevelDebug
	//level = LevelInfo
	level = LevelWarn

	// prefix for all messages, default is "[Gev]"
	prefix = "[zput]"
)

type defaultLogLogger struct{}

func (t *defaultLogLogger) Log(v ...interface{}) {
	stdlog.Print(v...)
}

func (t *defaultLogLogger) Logf(format string, v ...interface{}) {
	stdlog.Printf(format, v...)
}

func init() {
	switch os.Getenv("GEV_LOG_LEVEL") {
	case "fatal":
		level = LevelFatal
	case "error":
		level = LevelError
	case "warn":
		level = LevelWarn
	case "info":
		level = LevelInfo
	case "debug":
		level = LevelDebug
	}

	stdlog.SetFlags(stdlog.Ldate | stdlog.Lmicroseconds | stdlog.Lshortfile)
}

// Log makes use of Logger
func Log(l Level, v ...interface{}) {
	if len(prefix) > 0 {
		logger.Log(append([]interface{}{prefix, " ", l.String(), " "}, v...)...)
		return
	}
	logger.Log(v...)
}

// Logf makes use of Logger
func Logf(l Level, format string, v ...interface{}) {
	if len(prefix) > 0 {
		format = prefix + " " + l.String() + " " + format
	}
	logger.Logf(format, v...)
}

// WithLevel logs with the level specified
func WithLevel(l Level, v ...interface{}) {
	if l > level {
		return
	}
	Log(l, v...)
}

// WithLevelf logs with the level specified
func WithLevelf(l Level, format string, v ...interface{}) {
	if l > level {
		return
	}
	Logf(l, format, v...)
}

// Debug provides debug level logging
func Debug(v ...interface{}) {
	WithLevel(LevelDebug, v...)
}

// Debugf provides debug level logging
func Debugf(format string, v ...interface{}) {
	WithLevelf(LevelDebug, format, v...)
}

// Info provides info level logging
func Info(v ...interface{}) {
	WithLevel(LevelInfo, v...)
}

// Infof provides info level logging
func Infof(format string, v ...interface{}) {
	WithLevelf(LevelInfo, format, v...)
}

// Warn provides warn level logging
func Warn(v ...interface{}) {
	WithLevel(LevelWarn, v...)
}

// Warnf provides warn level logging
func Warnf(format string, v ...interface{}) {
	WithLevelf(LevelWarn, format, v...)
}

// Error provides warn level logging
func Error(v ...interface{}) {
	WithLevel(LevelError, v...)
}

// Errorf provides warn level logging
func Errorf(format string, v ...interface{}) {
	WithLevelf(LevelError, format, v...)
}

// Fatal logs with Log and then exits with os.Exit(1)
func Fatal(v ...interface{}) {
	WithLevel(LevelFatal, v...)
	os.Exit(1)
}

// Fatalf logs with Logf and then exits with os.Exit(1)
func Fatalf(format string, v ...interface{}) {
	WithLevelf(LevelFatal, format, v...)
	os.Exit(1)
}

// SetLogger sets the local logger
func SetLogger(l Logger) {
	logger = l
}

// GetLogger returns the local logger
func GetLogger() Logger {
	return logger
}

// SetLevel sets the log level
func SetLevel(l Level) {
	level = l
}

// GetLevel returns the current level
func GetLevel() Level {
	return level
}

// SetPrefix sets a prefix for the logger
func SetPrefix(p string) {
	prefix = p
}

// Name sets service name
func Name(name string) {
	prefix = fmt.Sprintf("[%s]", name)
}
