package models

import (
	"time"
)

// Request represents a logged API request
type Request struct {
	ID            int        `json:"id" db:"id"`
	UserID        *int       `json:"user_id,omitempty" db:"user_id" validate:"omitempty,gt=0"`
	SessionID     *int       `json:"session_id,omitempty" db:"session_id" validate:"omitempty,gt=0"`
	Method        string     `json:"method" db:"method" validate:"required,oneof=GET POST PUT DELETE PATCH OPTIONS HEAD"`
	Path          string     `json:"path" db:"path" validate:"required,max=500"`
	UserAgent     *string    `json:"user_agent,omitempty" db:"user_agent" validate:"omitempty,max=500"`
	IPAddress     string     `json:"ip_address" db:"ip_address" validate:"required,ip"`
	StatusCode    int        `json:"status_code" db:"status_code" validate:"required,gte=100,lte=599"`
	ResponseTime  int        `json:"response_time" db:"response_time" validate:"gte=0"` // in milliseconds
	RequestSize   int        `json:"request_size" db:"request_size" validate:"gte=0"`   // in bytes
	ResponseSize  int        `json:"response_size" db:"response_size" validate:"gte=0"` // in bytes
	Blocked       bool       `json:"blocked" db:"blocked"`
	BlockReason   *string    `json:"block_reason,omitempty" db:"block_reason" validate:"omitempty,max=255"`
	ContentType   *string    `json:"content_type,omitempty" db:"content_type" validate:"omitempty,max=100"`
	Referer       *string    `json:"referer,omitempty" db:"referer" validate:"omitempty,max=500"`
	RequestBody   *string    `json:"request_body,omitempty" db:"request_body" validate:"omitempty,max=10000"`
	ResponseBody  *string    `json:"response_body,omitempty" db:"response_body" validate:"omitempty,max=10000"`
	Headers       *string    `json:"headers,omitempty" db:"headers" validate:"omitempty,max=2000"` // JSON string
	ErrorMessage  *string    `json:"error_message,omitempty" db:"error_message" validate:"omitempty,max=1000"`
	Timestamp     time.Time  `json:"timestamp" db:"timestamp"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty" db:"processed_at"`
}

// RequestCreatePayload represents the payload for creating a request log
type RequestCreatePayload struct {
	UserID       *int    `json:"user_id,omitempty" validate:"omitempty,gt=0"`
	SessionID    *int    `json:"session_id,omitempty" validate:"omitempty,gt=0"`
	Method       string  `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH OPTIONS HEAD"`
	Path         string  `json:"path" validate:"required,max=500"`
	UserAgent    *string `json:"user_agent,omitempty" validate:"omitempty,max=500"`
	IPAddress    string  `json:"ip_address" validate:"required,ip"`
	StatusCode   int     `json:"status_code" validate:"required,gte=100,lte=599"`
	ResponseTime int     `json:"response_time" validate:"gte=0"`
	RequestSize  int     `json:"request_size" validate:"gte=0"`
	ResponseSize int     `json:"response_size" validate:"gte=0"`
	Blocked      bool    `json:"blocked"`
	BlockReason  *string `json:"block_reason,omitempty" validate:"omitempty,max=255"`
	ContentType  *string `json:"content_type,omitempty" validate:"omitempty,max=100"`
	Referer      *string `json:"referer,omitempty" validate:"omitempty,max=500"`
	RequestBody  *string `json:"request_body,omitempty" validate:"omitempty,max=10000"`
	ResponseBody *string `json:"response_body,omitempty" validate:"omitempty,max=10000"`
	Headers      *string `json:"headers,omitempty" validate:"omitempty,max=2000"`
	ErrorMessage *string `json:"error_message,omitempty" validate:"omitempty,max=1000"`
}

// RequestFilters represents filters for querying requests
type RequestFilters struct {
	UserID      *int       `json:"user_id,omitempty" validate:"omitempty,gt=0"`
	SessionID   *int       `json:"session_id,omitempty" validate:"omitempty,gt=0"`
	Method      *string    `json:"method,omitempty" validate:"omitempty,oneof=GET POST PUT DELETE PATCH OPTIONS HEAD"`
	Path        *string    `json:"path,omitempty" validate:"omitempty,max=500"`
	IPAddress   *string    `json:"ip_address,omitempty" validate:"omitempty,ip"`
	StatusCode  *int       `json:"status_code,omitempty" validate:"omitempty,gte=100,lte=599"`
	Blocked     *bool      `json:"blocked,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Limit       int        `json:"limit" validate:"gte=1,lte=1000"`
	Offset      int        `json:"offset" validate:"gte=0"`
	OrderBy     string     `json:"order_by" validate:"omitempty,oneof=timestamp status_code response_time"`
	OrderDir    string     `json:"order_dir" validate:"omitempty,oneof=asc desc"`
}

// RequestStats represents aggregated request statistics
type RequestStats struct {
	TotalRequests     int     `json:"total_requests"`
	BlockedRequests   int     `json:"blocked_requests"`
	SuccessRequests   int     `json:"success_requests"`
	ErrorRequests     int     `json:"error_requests"`
	AverageResponse   float64 `json:"average_response_time"`
	TotalResponseTime int64   `json:"total_response_time"`
	UniqueUsers       int     `json:"unique_users"`
	UniqueIPs         int     `json:"unique_ips"`
	Period            string  `json:"period"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
}

// IsBlocked checks if the request was blocked
func (r *Request) IsBlocked() bool {
	return r.Blocked
}

// IsSuccess checks if the request was successful (2xx status code)
func (r *Request) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsError checks if the request resulted in an error (4xx or 5xx status code)
func (r *Request) IsError() bool {
	return r.StatusCode >= 400
}

// GetResponseCategory returns a human-readable category for the response
func (r *Request) GetResponseCategory() string {
	switch {
	case r.IsBlocked():
		return "blocked"
	case r.StatusCode < 200:
		return "informational"
	case r.StatusCode < 300:
		return "success"
	case r.StatusCode < 400:
		return "redirect"
	case r.StatusCode < 500:
		return "client_error"
	default:
		return "server_error"
	}
}