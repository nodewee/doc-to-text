package logger

import (
	"fmt"
	"log"
	"os"
)

// LogLevel represents logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger provides structured logging
type Logger struct {
	level   LogLevel
	verbose bool
}

// NewLogger creates a new logger instance
func NewLogger(level string, verbose bool) *Logger {
	return &Logger{
		level:   parseLogLevel(level),
		verbose: verbose,
	}
}

// Debug logs debug messages
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", format, args...)
	}
}

// Info logs info messages
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", format, args...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		l.log("WARN", format, args...)
	}
}

// Error logs error messages
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= ERROR {
		l.log("ERROR", format, args...)
	}
}

// Progress logs progress messages with emoji
func (l *Logger) Progress(emoji, format string, args ...interface{}) {
	if l.verbose {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", emoji, message)
	}
}

// log is the internal logging method
func (l *Logger) log(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s", level, message)
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// DefaultLogger creates a default logger
func DefaultLogger() *Logger {
	return NewLogger("info", true)
}

// Fatal logs an error and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.Error(format, args...)
	os.Exit(1)
}
