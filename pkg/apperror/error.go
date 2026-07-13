package apperror

import (
	"errors"
	"fmt"
)

// Kind is a transport-independent classification of an application error.
// Services return these kinds; HTTP, NATS, and CLI adapters decide how to
// represent them on their own protocol boundary.
type Kind string

const (
	KindInvalidArgument  Kind = "invalid_argument"
	KindUnauthenticated  Kind = "unauthenticated"
	KindPermissionDenied Kind = "permission_denied"
	KindNotFound         Kind = "not_found"
	KindConflict         Kind = "conflict"
	KindRateLimited      Kind = "rate_limited"
	KindInternal         Kind = "internal"
	KindUnavailable      Kind = "unavailable"
)

type Error struct {
	kind    Kind
	message string
	cause   error
	details any
}

func New(kind Kind, message string, cause error) error {
	return &Error{kind: kind, message: message, cause: cause}
}

func WithDetails(kind Kind, message string, cause error, details any) error {
	return &Error{kind: kind, message: message, cause: cause, details: details}
}

func (e *Error) Error() string {
	if e.message != "" {
		return e.message
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return string(e.kind)
}

func (e *Error) Unwrap() error { return e.cause }
func (e *Error) Kind() Kind    { return e.kind }
func (e *Error) Message() string {
	return e.message
}
func (e *Error) Details() any { return e.details }

func InvalidArgument(message string, cause error) error {
	return New(KindInvalidArgument, message, cause)
}

func Unauthenticated(message string, cause error) error {
	return New(KindUnauthenticated, message, cause)
}

func PermissionDenied(message string, cause error) error {
	return New(KindPermissionDenied, message, cause)
}

func NotFound(message string, cause error) error { return New(KindNotFound, message, cause) }
func Conflict(message string, cause error) error { return New(KindConflict, message, cause) }
func RateLimited(message string, cause error) error {
	return New(KindRateLimited, message, cause)
}

func Internal(cause error) error { return New(KindInternal, "", cause) }

func Unavailable(message string, cause error) error {
	return New(KindUnavailable, message, cause)
}

func As(err error) (*Error, bool) {
	var target *Error
	if !errors.As(err, &target) {
		return nil, false
	}
	return target, true
}

func Wrap(kind Kind, message string, cause error) error {
	if cause == nil {
		cause = fmt.Errorf("%s", message)
	}
	return New(kind, message, cause)
}
