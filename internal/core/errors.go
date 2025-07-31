package core

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// AppError represents an application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Err.Error())
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error codes
const (
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeForbidden     = "FORBIDDEN"
	ErrCodeInternal      = "INTERNAL_ERROR"
	ErrCodeDatabase      = "DATABASE_ERROR"
	ErrCodeConfiguration = "CONFIGURATION_ERROR"
	ErrCodeFeature       = "FEATURE_ERROR"
)

// Common error constructors
func NewValidationError(message string, err error) *AppError {
	return NewAppError(ErrCodeValidation, message, err)
}

func NewNotFoundError(message string, err error) *AppError {
	return NewAppError(ErrCodeNotFound, message, err)
}

func NewUnauthorizedError(message string, err error) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, err)
}

func NewForbiddenError(message string, err error) *AppError {
	return NewAppError(ErrCodeForbidden, message, err)
}

func NewInternalError(message string, err error) *AppError {
	return NewAppError(ErrCodeInternal, message, err)
}

func NewDatabaseError(message string, err error) *AppError {
	return NewAppError(ErrCodeDatabase, message, err)
}

func NewConfigurationError(message string, err error) *AppError {
	return NewAppError(ErrCodeConfiguration, message, err)
}

func NewFeatureError(featureName, message string, err error) *AppError {
	return NewAppError(ErrCodeFeature, fmt.Sprintf("[%s] %s", featureName, message), err)
}

// ErrorResponse represents an error response for API endpoints
type ErrorResponse struct {
	Error   *AppError `json:"error"`
	Success bool      `json:"success"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err *AppError) *ErrorResponse {
	return &ErrorResponse{
		Error:   err,
		Success: false,
	}
}

// WriteErrorResponse writes an error response to an HTTP response writer
func WriteErrorResponse(w http.ResponseWriter, statusCode int, err *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := NewErrorResponse(err)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If we can't encode the error response, just write a simple error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GetHTTPStatusCode returns the appropriate HTTP status code for an error
func GetHTTPStatusCode(err *AppError) int {
	switch err.Code {
	case ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeInternal, ErrCodeDatabase, ErrCodeConfiguration, ErrCodeFeature:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// HandleError handles an error and writes an appropriate HTTP response
func HandleError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if e, ok := err.(*AppError); ok {
		appErr = e
	} else {
		// Convert generic error to internal error
		appErr = NewInternalError("An unexpected error occurred", err)
	}

	statusCode := GetHTTPStatusCode(appErr)
	WriteErrorResponse(w, statusCode, appErr)
}
