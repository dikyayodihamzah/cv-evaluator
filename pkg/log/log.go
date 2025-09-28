package log

import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

func Error(message string) {
	logger.Printf("[ERROR] %s", message)
}

func Info(message string) {
	logger.Printf("[INFO] %s", message)
}

func Debug(message string) {
	logger.Printf("[DEBUG] %s", message)
}

func Warn(message string) {
	logger.Printf("[WARN] %s", message)
}

func Fatal(message string) {
	logger.Fatalf("[FATAL] %s", message)
}
