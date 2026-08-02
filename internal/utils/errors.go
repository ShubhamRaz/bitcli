// Package utils contains shared helpers that are intentionally small and dependency-light.
package utils

import (
	"errors"
	"fmt"
)

// Code identifies stable error classes for CLI and API callers.
type Code string

const (
	CodeModelNotFound             Code = "model_not_found"
	CodeBackendNotFound           Code = "backend_not_found"
	CodeBackendUnsupportedOption  Code = "backend_unsupported_option"
	CodeDownloadInterrupted       Code = "download_interrupted"
	CodeChecksumMismatch          Code = "checksum_mismatch"
	CodeConfigInvalid             Code = "config_invalid"
	CodeInsufficientDisk          Code = "insufficient_disk"
	CodeHardwareUnsupported       Code = "hardware_unsupported"
	CodeInvalidInput              Code = "invalid_input"
	CodeUnavailable               Code = "unavailable"
	CodeInternal                  Code = "internal"
)

// Error carries a stable code plus a human-readable message.
type Error struct {
	Code Code
	Msg  string
	Err  error
}

// Error renders the error message.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Msg
	}
	if e.Msg == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Msg, e.Err)
}

// Unwrap exposes the wrapped cause.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewError creates a coded error without a wrapped cause.
func NewError(code Code, msg string) error {
	return &Error{Code: code, Msg: msg}
}

// WrapError creates a coded error with a wrapped cause.
func WrapError(code Code, msg string, err error) error {
	if err == nil {
		return NewError(code, msg)
	}
	return &Error{Code: code, Msg: msg, Err: err}
}

// ErrorCode returns the stable code for an error, defaulting to internal.
func ErrorCode(err error) Code {
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	return CodeInternal
}

