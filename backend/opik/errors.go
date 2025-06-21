package opik

import "errors"

var (
	// ErrMissingAPIKey is returned when API key is not provided
	ErrMissingAPIKey = errors.New("opik: API key is required when enabled")
	
	// ErrClientClosed is returned when operations are attempted on a closed client
	ErrClientClosed = errors.New("opik: client is closed")
	
	// ErrInvalidSpan is returned when span operations fail
	ErrInvalidSpan = errors.New("opik: invalid span")
	
	// ErrInvalidTrace is returned when trace operations fail
	ErrInvalidTrace = errors.New("opik: invalid trace")
)