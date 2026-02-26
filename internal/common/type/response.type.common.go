package types

import "time"

type Response struct {
	Data    any
	Message string
	Code    int
	Error   error
}

type ResponseAPIDebug struct {
	RequestID string    `json:"requestId"`
	Version   string    `json:"version"`
	Error     *string   `json:"error"`
	StartTime time.Time `json:"startTime"` // ISO8601 format, e.g., "2025-01-09T15:04:05Z07:00"
	EndTime   time.Time `json:"endTime"`   // ISO8601 format for consistency with StartTime
	RuntimeMs int64     `json:"runtimeMs"` // Runtime in milliseconds for better precision
}

type ResponseAPI struct {
	Data    any               `json:"data"`
	Message string            `json:"message"`
	Debug   *ResponseAPIDebug `json:"debug,omitempty"`
}
