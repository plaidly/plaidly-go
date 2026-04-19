package plaidly

import "fmt"

// Error is returned when the Plaidly API responds with a non-2xx status code.
type Error struct {
	// StatusCode is the HTTP status code returned by the API.
	StatusCode int
	// Code is the machine-readable error code from the API response body.
	Code string
	// Message is the human-readable error message from the API response body.
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("plaidly: %s (code=%s, status=%d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("plaidly: HTTP %d (code=%s)", e.StatusCode, e.Code)
}
