package observability

import (
	"context"
	"log"
	"os"
)

// Logger provides structured logging
type Logger struct {
	logger *log.Logger
	level  string
}

// NewLogger creates a new logger
func NewLogger(level string) *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
		level:  level,
	}
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.logger.Printf("[INFO] %s %v", msg, fields)
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, msg string, err error, fields ...interface{}) {
	l.logger.Printf("[ERROR] %s: %v %v", msg, err, fields)
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	if l.level == "debug" {
		l.logger.Printf("[DEBUG] %s %v", msg, fields)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	l.logger.Printf("[WARN] %s %v", msg, fields)
}
