package lib

import (
	"runtime"
	"time"
)

type Response struct {
	Code             int         `json:"code"`
	Status           bool        `json:"status"`
	Message          string      `json:"message"`
	Data             interface{} `json:"data,omitempty"`
	ErrorDescription *string     `json:"error_description,omitempty"`
	Log              *LogInfo    `json:"log,omitempty"`
}

func (r *Response) Error() string {
	return r.Message
}

type LogInfo struct {
	Timestamp string `json:"timestamp"`
	File      string `json:"file"`
	Line      int    `json:"line"`
}

func Pointer(s string) *string {
	return &s
}

func AddLog() *LogInfo {
	_, file, line, _ := runtime.Caller(2)
	return &LogInfo{
		Timestamp: time.Now().Format(time.RFC3339),
		File:      file,
		Line:      line,
	}
}

var ValidateMessages = map[string]string{
	"required": "%s is required",
	"email":    "%s must be a valid email",
	"min":      "%s must be at least %s characters",
	"max":      "%s must be at most %s characters",
}
