// Logging helpers
//
package longshoreman

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"fmt"
	style "github.com/azer/go-style"
	"log"
	"os"
	"syscall"
)

const (
	LogLevelDebug = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var logLevelNames = map[uint]string{
	LogLevelDebug: "DEBUG",
	LogLevelInfo:  " INFO",
	LogLevelWarn:  " WARN",
	LogLevelError: "ERROR",
}

var logLevelStyles = map[uint]string{
	LogLevelDebug: "inverse",
	LogLevelInfo:  "bold",
	LogLevelWarn:  "yellow",
	LogLevelError: "red blink",
}

type Logger struct {
	*log.Logger
	LogLevel uint
	Colorize bool // if terminal is detected
}

func NewLogger(level uint) *Logger {
	return &Logger{log.New(os.Stdout, "", log.LstdFlags), level, true}
}

func (l *Logger) logWithLevel(level uint, format string, args ...interface{}) {
	if level < l.LogLevel {
		return
	}

	if l.Colorize && terminal.IsTerminal(syscall.Stdout) {
		l.Printf("[%s%s] %s\n",
			style.Style(logLevelStyles[level], logLevelNames[level]),
			style.Style("reset", ""),
			fmt.Sprintf(format, args...))
	} else {
		l.Printf("[%s] %s\n", logLevelNames[level], fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.logWithLevel(LogLevelDebug, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.logWithLevel(LogLevelInfo, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.logWithLevel(LogLevelError, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.logWithLevel(LogLevelWarn, format, args...)
}
