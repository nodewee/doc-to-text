package logger

import (
	"fmt"
	"log"
	"os"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger provides structured logging functionality
type Logger struct {
	level   LogLevel
	verbose bool
}

// NewLogger creates a new logger with specified level and verbose mode
func NewLogger(level string, verbose bool) *Logger {
	return &Logger{
		level:   parseLogLevel(level),
		verbose: verbose,
	}
}

// Debug logs debug information (only in debug mode)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= LevelDebug {
		message := fmt.Sprintf(format, args...)
		l.log("DEBUG", message)
	}
}

// Info logs informational messages (only in verbose mode)
func (l *Logger) Info(format string, args ...interface{}) {
	if l.verbose && l.level <= LevelInfo {
		message := fmt.Sprintf(format, args...)
		l.log("INFO", message)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		message := fmt.Sprintf(format, args...)
		l.log("WARN", message)
	}
}

// Error logs error messages
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LevelError {
		message := fmt.Sprintf(format, args...)
		l.log("ERROR", message)
	}
}

// ProgressAlways logs critical progress information that should always be shown
// This is for important milestones that users should see regardless of verbose mode
func (l *Logger) ProgressAlways(emoji, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", emoji, message)
}

// Progress logs detailed progress information (only in verbose mode)
// This is for step-by-step details that help with debugging and monitoring
func (l *Logger) Progress(emoji, format string, args ...interface{}) {
	if l.verbose {
		message := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", emoji, message)
	}
}

// log outputs formatted log messages
func (l *Logger) log(level, message string) {
	fmt.Printf("[%s] %s\n", level, message)
}

// parseLogLevel converts string level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// DefaultLogger returns a default logger instance
func DefaultLogger() *Logger {
	return NewLogger("info", false)
}

// Fatal logs a fatal error and exits the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Fatalf("FATAL: %s", message)
	os.Exit(1)
}
