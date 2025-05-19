package utils

import (
	"fmt"

	"github.com/zinrai/ansible-template-render/internal/logger"
)

type ErrorCode int

const (
	// Error code definitions
	ErrUnknown ErrorCode = iota
	ErrFileNotFound
	ErrInvalidConfig
	ErrAnsibleExecution
	ErrRoleProcessing
)

// Represents an application error with context
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Returns the error message
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// Creates a new application error
func NewError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Logs an error and returns it
func LogAndReturnError(err error) error {
	logger.Error("Error occurred", "error", err)
	return err
}

// Logs a warning and returns nil to allow continuing
func LogWarningAndContinue(message string, err error) {
	logger.Warn(message, "error", err)
}
