package vectorstore

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger interface defines the logging methods required by components
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// SimpleLogger is a basic implementation of the Logger interface
type SimpleLogger struct {
	level  LogLevel
	prefix string
	logger *log.Logger
}

// NewSimpleLogger creates a new logger with the specified minimum level
func NewSimpleLogger(level LogLevel, prefix string) *SimpleLogger {
	return &SimpleLogger{
		level:  level,
		prefix: prefix,
		logger: log.New(os.Stdout, "", 0), // No built-in timestamp to avoid duplication
	}
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(format string, args ...interface{}) {
	if l.level <= DebugLevel {
		l.log("DEBUG", format, args...)
	}
}

// Info logs an informational message
func (l *SimpleLogger) Info(format string, args ...interface{}) {
	if l.level <= InfoLevel {
		l.log("INFO", format, args...)
	}
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(format string, args ...interface{}) {
	if l.level <= WarnLevel {
		l.log("WARN", format, args...)
	}
}

// Error logs an error message
func (l *SimpleLogger) Error(format string, args ...interface{}) {
	if l.level <= ErrorLevel {
		l.log("ERROR", format, args...)
	}
}

// log formats and writes a log message
func (l *SimpleLogger) log(level string, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("%s [%s] %s: %s", timestamp, level, l.prefix, message)
}

// NullLogger implements Logger but discards all messages
type NullLogger struct{}

func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

func (l *NullLogger) Debug(format string, args ...interface{}) {}
func (l *NullLogger) Info(format string, args ...interface{})  {}
func (l *NullLogger) Warn(format string, args ...interface{})  {}
func (l *NullLogger) Error(format string, args ...interface{}) {}