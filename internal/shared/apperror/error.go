package apperror

import (
	"fmt"
	"net/http"
)

type ErrorType string

const (
	TypeBadRequest     ErrorType = "BAD_REQUEST"
	TypeUnauthorized   ErrorType = "UNAUTHORIZED"
	TypeForbidden      ErrorType = "FORBIDDEN"
	TypeNotFound       ErrorType = "NOT_FOUND"
	TypeConflict       ErrorType = "CONFLICT"
	TypeValidation     ErrorType = "VALIDATION_ERROR"
	TypeQuotaExceeded  ErrorType = "QUOTA_EXCEEDED"
	TypeInternalServer ErrorType = "INTERNAL_SERVER_ERROR"
)

type AppError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Details any       `json:"details,omitempty"`
	Err     error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func (e *AppError) HTTPStatus() int {
	switch e.Type {
	case TypeBadRequest, TypeValidation:
		return http.StatusBadRequest
	case TypeUnauthorized:
		return http.StatusUnauthorized
	case TypeForbidden:
		return http.StatusForbidden
	case TypeNotFound:
		return http.StatusNotFound
	case TypeConflict:
		return http.StatusConflict
	case TypeQuotaExceeded:
		return http.StatusRequestEntityTooLarge
	case TypeInternalServer:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func New(errType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

func BadRequest(message string, err error) *AppError {
	return New(TypeBadRequest, message, err)
}

func Unauthorized(message string) *AppError {
	return New(TypeUnauthorized, message, nil)
}

func Forbidden(message string) *AppError {
	return New(TypeForbidden, message, nil)
}

func NotFound(message string) *AppError {
	return New(TypeNotFound, message, nil)
}

func Conflict(message string) *AppError {
	return New(TypeConflict, message, nil)
}

func Validation(message string, details any) *AppError {
	return &AppError{
		Type:    TypeValidation,
		Message: message,
		Details: details,
	}
}

func QuotaExceeded(message string) *AppError {
	return New(TypeQuotaExceeded, message, nil)
}

func Internal(message string, err error) *AppError {
	return New(TypeInternalServer, message, err)
}
