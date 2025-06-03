package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation  ErrorType = "validation"
	ErrorTypeIO          ErrorType = "io"
	ErrorTypeNetwork     ErrorType = "network"
	ErrorTypeOCR         ErrorType = "ocr"
	ErrorTypeConversion  ErrorType = "conversion"
	ErrorTypeSystem      ErrorType = "system"
	ErrorTypeUnsupported ErrorType = "unsupported"
	ErrorTypeTimeout     ErrorType = "timeout"
	ErrorTypePermission  ErrorType = "permission"
	ErrorTypeNotFound    ErrorType = "not_found"
)

// AppError represents an application-specific error with context
type AppError struct {
	Type        ErrorType
	Message     string
	Cause       error
	Context     map[string]interface{}
	Recoverable bool
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Type == t.Type
	}
	return false
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewError creates a new application error
func NewError(errorType ErrorType, message string, cause error) *AppError {
	return &AppError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string, cause error) *AppError {
	return NewError(ErrorTypeValidation, message, cause)
}

// NewIOError creates an I/O error
func NewIOError(message string, cause error) *AppError {
	return NewError(ErrorTypeIO, message, cause)
}

// NewOCRError creates an OCR error
func NewOCRError(message string, cause error) *AppError {
	return NewError(ErrorTypeOCR, message, cause)
}

// NewConversionError creates a conversion error
func NewConversionError(message string, cause error) *AppError {
	return NewError(ErrorTypeConversion, message, cause)
}

// NewUnsupportedError creates an unsupported operation error
func NewUnsupportedError(message string, cause error) *AppError {
	return NewError(ErrorTypeUnsupported, message, cause)
}

// NewSystemError creates a system error
func NewSystemError(message string, cause error) *AppError {
	return NewError(ErrorTypeSystem, message, cause)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(message string, cause error) *AppError {
	return NewError(ErrorTypeNotFound, message, cause)
}

// NewPermissionError creates a permission error
func NewPermissionError(message string, cause error) *AppError {
	return NewError(ErrorTypePermission, message, cause)
}

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, message string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, preserve the original type unless explicitly overridden
	if appErr, ok := err.(*AppError); ok && errorType == "" {
		return &AppError{
			Type:    appErr.Type,
			Message: message + ": " + appErr.Message,
			Cause:   appErr.Cause,
			Context: appErr.Context,
		}
	}

	if errorType == "" {
		errorType = classifyError(err)
	}

	return &AppError{
		Type:    errorType,
		Message: message,
		Cause:   err,
		Context: make(map[string]interface{}),
	}
}

// classifyError automatically classifies an error based on its content
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeSystem
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return ErrorTypeTimeout
	case strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "access denied"):
		return ErrorTypePermission
	case strings.Contains(errStr, "no such file") || strings.Contains(errStr, "not found"):
		return ErrorTypeNotFound
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection"):
		return ErrorTypeNetwork
	case strings.Contains(errStr, "ocr") || strings.Contains(errStr, "extraction"):
		return ErrorTypeOCR
	case strings.Contains(errStr, "convert") || strings.Contains(errStr, "parsing"):
		return ErrorTypeConversion
	case strings.Contains(errStr, "invalid") || strings.Contains(errStr, "bad"):
		return ErrorTypeValidation
	default:
		return ErrorTypeSystem
	}
}

// IsRecoverable checks if an error is recoverable
func IsRecoverable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Recoverable
	}

	// Default recovery rules based on error type
	errorType := classifyError(err)
	switch errorType {
	case ErrorTypeTimeout, ErrorTypeNetwork:
		return true
	case ErrorTypePermission, ErrorTypeNotFound, ErrorTypeValidation:
		return false
	default:
		return false
	}
}

// GetErrorType extracts the error type from an error
func GetErrorType(err error) ErrorType {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type
	}
	return classifyError(err)
}

// RecoveryAction represents an action to take for error recovery
type RecoveryAction func(err error) error

// ErrorHandler provides centralized error handling with recovery strategies
type ErrorHandler struct {
	recoveryStrategies map[ErrorType]RecoveryAction
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		recoveryStrategies: make(map[ErrorType]RecoveryAction),
	}
}

// RegisterRecoveryStrategy registers a recovery strategy for a specific error type
func (eh *ErrorHandler) RegisterRecoveryStrategy(errorType ErrorType, action RecoveryAction) {
	eh.recoveryStrategies[errorType] = action
}

// Handle handles an error with optional recovery
func (eh *ErrorHandler) Handle(err error, attemptRecovery bool) error {
	if err == nil {
		return nil
	}

	errorType := GetErrorType(err)

	if attemptRecovery && IsRecoverable(err) {
		if strategy, exists := eh.recoveryStrategies[errorType]; exists {
			if recoveryErr := strategy(err); recoveryErr == nil {
				return nil // Recovery successful
			}
		}
	}

	return err
}

// WithRetry executes a function with retry logic for recoverable errors
func WithRetry(fn func() error, maxAttempts int, eh *ErrorHandler) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if !IsRecoverable(err) {
			return err // Don't retry non-recoverable errors
		}

		if attempt < maxAttempts {
			// Try recovery if handler is provided
			if eh != nil {
				if recoveredErr := eh.Handle(err, true); recoveredErr == nil {
					continue // Recovery successful, retry
				}
			}
		}
	}

	return WrapError(lastErr, "", fmt.Sprintf("operation failed after %d attempts", maxAttempts))
}
