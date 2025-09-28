package log

import (
	"fmt"
	"log"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

func Error(message string, args ...interface{}) {
	logger.Printf("[ERROR] %s", fmt.Sprintf(message, args...))
}

func Info(message string, args ...interface{}) {
	logger.Printf("[INFO] %s", fmt.Sprintf(message, args...))
}

func Debug(message string, args ...interface{}) {
	logger.Printf("[DEBUG] %s", fmt.Sprintf(message, args...))
}

func Warn(message string, args ...interface{}) {
	logger.Printf("[WARN] %s", fmt.Sprintf(message, args...))
}

func Fatal(message string, args ...interface{}) {
	logger.Fatalf("[FATAL] %s", fmt.Sprintf(message, args...))
}

// WithContext adds contextual information to logs
func WithContext(component, operation string) *ContextLogger {
	return &ContextLogger{
		component: component,
		operation: operation,
	}
}

type ContextLogger struct {
	component string
	operation string
}

func (c *ContextLogger) Error(message string, args ...interface{}) {
	logger.Printf("[ERROR] [%s:%s] %s", c.component, c.operation, fmt.Sprintf(message, args...))
}

func (c *ContextLogger) Info(message string, args ...interface{}) {
	logger.Printf("[INFO] [%s:%s] %s", c.component, c.operation, fmt.Sprintf(message, args...))
}

func (c *ContextLogger) Debug(message string, args ...interface{}) {
	logger.Printf("[DEBUG] [%s:%s] %s", c.component, c.operation, fmt.Sprintf(message, args...))
}

func (c *ContextLogger) Warn(message string, args ...interface{}) {
	logger.Printf("[WARN] [%s:%s] %s", c.component, c.operation, fmt.Sprintf(message, args...))
}

func (c *ContextLogger) WithDuration(start time.Time) *ContextLogger {
	duration := time.Since(start)
	c.Info("Operation completed in %v", duration)
	return c
}
