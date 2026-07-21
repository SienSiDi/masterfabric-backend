package errors

import (
	stderrors "errors"
)

type Code string

const (
	CodeNotFound     Code = "NOT_FOUND"
	CodeBadRequest   Code = "INVALID_INPUT"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeForbidden    Code = "FORBIDDEN"
	CodeConflict     Code = "CONFLICT"
	CodeTooMany      Code = "RATE_LIMITED"
	CodeInternal     Code = "INTERNAL"
)

var (
	ErrNotFound     = New(CodeNotFound, "resource not found", nil)
	ErrBadRequest   = New(CodeBadRequest, "invalid input", nil)
	ErrUnauthorized = New(CodeUnauthorized, "unauthorized", nil)
	ErrForbidden    = New(CodeForbidden, "forbidden", nil)
	ErrConflict     = New(CodeConflict, "conflict", nil)
	ErrTooMany      = New(CodeTooMany, "rate limit exceeded", nil)
	ErrInternal     = New(CodeInternal, "an internal error occurred", nil)
)

type DomainError struct {
	Code    Code
	Message string
	Cause   error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.Cause.Error()
	}
	return string(e.Code) + ": " + e.Message
}

func (e *DomainError) Unwrap() error { return e.Cause }

func New(code Code, message string, cause error) *DomainError {
	return &DomainError{Code: code, Message: message, Cause: cause}
}

func As(err error) (*DomainError, bool) {
	var de *DomainError
	if ok := stderrors.As(err, &de); ok {
		return de, true
	}
	return nil, false
}
