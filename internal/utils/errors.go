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
	ErrTaskProcessing
	ErrTemplateProcessing
	ErrEnvironmentSetup
	ErrExternalDependency
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

// Returns true if the error is a "not found" type error
func IsNotFoundError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrFileNotFound
	}
	return false
}

// Returns true if the error is related to configuration
func IsConfigError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrInvalidConfig
	}
	return false
}

// Creates a new application error
func NewError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Creates a file not found error
func NewFileNotFoundError(path string, err error) *AppError {
	message := fmt.Sprintf("file not found: %s", path)
	return NewError(ErrFileNotFound, message, err)
}

// Creates a configuration error
func NewConfigError(message string, err error) *AppError {
	return NewError(ErrInvalidConfig, message, err)
}

// Creates an Ansible execution error
func NewAnsibleExecutionError(message string, err error) *AppError {
	return NewError(ErrAnsibleExecution, message, err)
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
