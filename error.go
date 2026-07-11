package plaidly

import (
	"errors"
	"fmt"
)

const (
	ErrorCodeInternal        int64 = 1
	ErrorCodeNotFound        int64 = 2
	ErrorCodeBadRequest      int64 = 3
	ErrorCodeTooManyRequests int64 = 4
	ErrorCodeConflict        int64 = 5
	ErrorCodeClientCancelled int64 = 6
	ErrorCodeForbidden       int64 = 7
	ErrorCodeUnavailable     int64 = 8
)

type Error struct {
	StatusCode int
	Code       int64
	Message    string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("plaidly: %s (code=%d, status=%d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("plaidly: HTTP %d (code=%d)", e.StatusCode, e.Code)
}

func (e *Error) IsConflict() bool {
	return e.Code == ErrorCodeConflict
}

func (e *Error) IsIdempotencyConflict() bool {
	return e.StatusCode == 409 && e.Code == ErrorCodeConflict
}

func IsIdempotencyConflict(err error) bool {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.IsIdempotencyConflict()
}
