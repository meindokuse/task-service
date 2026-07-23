// Package terror определяет типизированные доменные ошибки с привязанным
// HTTP-статусом, чтобы presentation-слой мог сопоставлять их без знания деталей бизнес-логики.
package terror

import (
	"errors"
	"net/http"
)

type Kind int

const (
	KindInternal Kind = iota
	KindNotFound
	KindConflict
	KindForbidden
	KindUnauthorized
	KindValidation
)

type Error struct {
	Kind    Kind
	Message string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, cause error) *Error {
	return &Error{Kind: kind, Message: message, cause: cause}
}

func NotFound(message string) *Error     { return New(KindNotFound, message) }
func Conflict(message string) *Error     { return New(KindConflict, message) }
func Forbidden(message string) *Error    { return New(KindForbidden, message) }
func Unauthorized(message string) *Error { return New(KindUnauthorized, message) }
func Validation(message string) *Error   { return New(KindValidation, message) }
func Internal(message string, cause error) *Error {
	return Wrap(KindInternal, message, cause)
}

// HTTPStatus сопоставляет (возможно, обёрнутую) ошибку с HTTP-статусом.
func HTTPStatus(err error) int {
	var e *Error
	if errors.As(err, &e) {
		switch e.Kind {
		case KindNotFound:
			return http.StatusNotFound
		case KindConflict:
			return http.StatusConflict
		case KindForbidden:
			return http.StatusForbidden
		case KindUnauthorized:
			return http.StatusUnauthorized
		case KindValidation:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}
