package logger

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fatih/color"
)

type Logger struct {
	serviceName string
}

var (
	// Emoji constants
	INFO_EMOJI    = "ℹ️ "
	SUCCESS_EMOJI = "✅ "
	WARN_EMOJI    = "⚠️ "
	ERROR_EMOJI   = "❌ "
	DEBUG_EMOJI   = "🔍 "
)

func New(serviceName string) *Logger {
	return &Logger{
		serviceName: serviceName,
	}
}

func (l *Logger) formatMessage(level, emoji, msg string) string {
	_, file, line, _ := runtime.Caller(2)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fileName := filepath.Base(file)

	return fmt.Sprintf("%s | %s | %s | %s:%d | %s | %s",
		emoji,
		timestamp,
		level,
		fileName,
		line,
		l.serviceName,
		msg,
	)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	formatted := l.formatMessage("INFO", INFO_EMOJI, fmt.Sprintf(msg, args...))
	color.Cyan(formatted)
}

func (l *Logger) Success(msg string, args ...interface{}) {
	formatted := l.formatMessage("SUCCESS", SUCCESS_EMOJI, fmt.Sprintf(msg, args...))
	color.Green(formatted)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	formatted := l.formatMessage("WARN", WARN_EMOJI, fmt.Sprintf(msg, args...))
	color.Yellow(formatted)
}

func (l *Logger) Error(msg string, err error, args ...interface{}) error {
	args = append(args, err)
	formatted := l.formatMessage("ERROR", ERROR_EMOJI, fmt.Sprintf(msg, args...))
	color.Red(formatted)
	return fmt.Errorf("%s: %w", msg, err)
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	formatted := l.formatMessage("DEBUG", DEBUG_EMOJI, fmt.Sprintf(msg, args...))
	color.Magenta(formatted)
}
