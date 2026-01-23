package errors

import (
	"errors"
	"fmt"
)

// Error codes
const (
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeUserExists        = "USER_EXISTS"
	CodeNotFound          = "NOT_FOUND"
	CodeForbidden         = "FORBIDDEN"
	CodeInvalidInput      = "INVALID_INPUT"
	CodeValidationError   = "VALIDATION_ERROR"
	CodeInternalError     = "INTERNAL_ERROR"
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	CodeBadRequest        = "BAD_REQUEST"
)

// AppError represents a structured application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError
func NewAppError(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewAppErrorWithDetails creates a new AppError with details
func NewAppErrorWithDetails(code, message, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapAppError wraps an existing error in an AppError
func WrapAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Details: err.Error(),
	}
}

// Predefined errors
var (
	ErrUnauthorized      = NewAppError(CodeUnauthorized, "Unauthorized access")
	ErrInvalidCredentials = NewAppError(CodeInvalidCredentials, "Invalid credentials")
	ErrUserExists        = NewAppError(CodeUserExists, "User already exists")
	ErrNotFound          = NewAppError(CodeNotFound, "Resource not found")
	ErrForbidden         = NewAppError(CodeForbidden, "Access forbidden")
	ErrInvalidInput      = NewAppError(CodeInvalidInput, "Invalid input provided")
	ErrRateLimitExceeded = NewAppError(CodeRateLimitExceeded, "Rate limit exceeded")
	ErrBadRequest        = NewAppError(CodeBadRequest, "Bad request")
)

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts AppError from error chain
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}