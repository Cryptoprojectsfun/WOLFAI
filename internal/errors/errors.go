package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents the type of error
type ErrorType uint

const (
	// Error types
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeValidation
	ErrorTypeAuthentication
	ErrorTypeAuthorization
	ErrorTypeNotFound
	ErrorTypeConflict
	ErrorTypeInternal
	ErrorTypeExternal
	ErrorTypeRateLimit
)

// Error represents a custom error with additional context
type Error struct {
	Type       ErrorType
	Message    string
	Details    map[string]interface{}
	Err        error
	StatusCode int
	ErrorCode  string
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new custom error
func NewError(errType ErrorType, message string, err error) *Error {
	statusCode := errorTypeToStatusCode(errType)
	errorCode := errorTypeToCode(errType)

	return &Error{
		Type:       errType,
		Message:    message,
		Err:        err,
		StatusCode: statusCode,
		ErrorCode:  errorCode,
		Details:    make(map[string]interface{}),
	}
}

// WithDetails adds context details to the error
func (e *Error) WithDetails(details map[string]interface{}) *Error {
	e.Details = details
	return e
}

// Is implements error comparison
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// Common error constructors
func NewValidationError(message string, err error) *Error {
	return NewError(ErrorTypeValidation, message, err)
}

func NewAuthenticationError(message string, err error) *Error {
	return NewError(ErrorTypeAuthentication, message, err)
}

func NewAuthorizationError(message string, err error) *Error {
	return NewError(ErrorTypeAuthorization, message, err)
}

func NewNotFoundError(message string, err error) *Error {
	return NewError(ErrorTypeNotFound, message, err)
}

func NewConflictError(message string, err error) *Error {
	return NewError(ErrorTypeConflict, message, err)
}

func NewInternalError(message string, err error) *Error {
	return NewError(ErrorTypeInternal, message, err)
}

func NewExternalError(message string, err error) *Error {
	return NewError(ErrorTypeExternal, message, err)
}

func NewRateLimitError(message string, err error) *Error {
	return NewError(ErrorTypeRateLimit, message, err)
}

// Helper functions
func errorTypeToStatusCode(errType ErrorType) int {
	switch errType {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case ErrorTypeAuthorization:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	case ErrorTypeExternal:
		return http.StatusBadGateway
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func errorTypeToCode(errType ErrorType) string {
	switch errType {
	case ErrorTypeValidation:
		return "VALIDATION_ERROR"
	case ErrorTypeAuthentication:
		return "AUTHENTICATION_ERROR"
	case ErrorTypeAuthorization:
		return "AUTHORIZATION_ERROR"
	case ErrorTypeNotFound:
		return "NOT_FOUND"
	case ErrorTypeConflict:
		return "CONFLICT"
	case ErrorTypeInternal:
		return "INTERNAL_ERROR"
	case ErrorTypeExternal:
		return "EXTERNAL_ERROR"
	case ErrorTypeRateLimit:
		return "RATE_LIMIT_EXCEEDED"
	default:
		return "UNKNOWN_ERROR"
	}
}

// Domain-specific error constructors
func NewPortfolioNotFoundError(portfolioID string) *Error {
	return NewNotFoundError(
		fmt.Sprintf("Portfolio not found: %s", portfolioID),
		nil,
	).WithDetails(map[string]interface{}{
		"portfolio_id": portfolioID,
	})
}

func NewInvalidPortfolioError(message string, validationErrors map[string]string) *Error {
	return NewValidationError(
		message,
		nil,
	).WithDetails(map[string]interface{}{
		"validation_errors": validationErrors,
	})
}

func NewMarketDataError(symbol string, err error) *Error {
	return NewExternalError(
		fmt.Sprintf("Failed to fetch market data for %s", symbol),
		err,
	).WithDetails(map[string]interface{}{
		"symbol": symbol,
	})
}

func NewModelPredictionError(modelID string, err error) *Error {
	return NewInternalError(
		fmt.Sprintf("Model prediction failed for %s", modelID),
		err,
	).WithDetails(map[string]interface{}{
		"model_id": modelID,
	})
}

func NewDatabaseError(operation string, err error) *Error {
	return NewInternalError(
		fmt.Sprintf("Database operation failed: %s", operation),
		err,
	).WithDetails(map[string]interface{}{
		"operation": operation,
	})
}

func NewInvalidCredentialsError() *Error {
	return NewAuthenticationError(
		"Invalid credentials",
		nil,
	)
}

func NewTokenExpiredError() *Error {
	return NewAuthenticationError(
		"Token has expired",
		nil,
	)
}

func NewInvalidTokenError() *Error {
	return NewAuthenticationError(
		"Invalid token",
		nil,
	)
}

func NewRateLimitExceededError(limit int, windowSeconds int) *Error {
	return NewRateLimitError(
		fmt.Sprintf("Rate limit exceeded: %d requests per %d seconds", limit, windowSeconds),
		nil,
	).WithDetails(map[string]interface{}{
		"limit":         limit,
		"window_secs":   windowSeconds,
		"retry_after":   windowSeconds,
	})
}

// Error Response structure for API responses
type ErrorResponse struct {
	Status     string                 `json:"status"`
	ErrorCode  string                 `json:"error_code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Timestamp  int64                  `json:"timestamp"`
}

// NewErrorResponse creates an error response from an Error
func NewErrorResponse(err *Error, requestID string) *ErrorResponse {
	return &ErrorResponse{
		Status:     "error",
		ErrorCode:  err.ErrorCode,
		Message:    err.Message,
		Details:    err.Details,
		RequestID:  requestID,
		Timestamp:  time.Now().Unix(),
	}
}