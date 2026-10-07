// Package apperr define los tipos de error del microservicio y su conversión
// al formato RFC 9457 (Problem Details for HTTP APIs).
package apperr

import (
	"errors"
	"fmt"

	"microservicio-io/internal/model"
)

// Code es el identificador estable del problema, tal como lo espera el monolito.
type Code string

const (
	CodeInvalidFile  Code = "invalid-file"
	CodeMalformedPDF Code = "malformed-pdf"
	CodeTooLarge     Code = "too-large"
	CodeTimeout      Code = "timeout"
	CodeServer       Code = "server-error"
)

// AppError es el error de dominio del microservicio.
type AppError struct {
	Code   Code
	Detail string
}

func (e *AppError) Error() string { return e.Detail }

func InvalidFile(detail string) *AppError {
	return &AppError{Code: CodeInvalidFile, Detail: detail}
}

func MalformedPDF(detail string) *AppError {
	return &AppError{Code: CodeMalformedPDF, Detail: detail}
}

func TooLarge(maxMB int64) *AppError {
	return &AppError{
		Code:   CodeTooLarge,
		Detail: fmt.Sprintf("El archivo excede el límite permitido de %d MB.", maxMB),
	}
}

func Timeout() *AppError {
	return &AppError{Code: CodeTimeout, Detail: "Tiempo de extracción de texto agotado."}
}

func Server(detail string) *AppError {
	return &AppError{Code: CodeServer, Detail: detail}
}

// statusFor devuelve el código HTTP asociado a cada Code (RFC 9457).
func statusFor(code Code) int {
	switch code {
	case CodeInvalidFile:
		return 400
	case CodeMalformedPDF:
		return 422
	case CodeTooLarge:
		return 413
	case CodeTimeout:
		return 504
	default:
		return 500
	}
}

// CodeOf extrae el Code de un error; devuelve server-error si no es un AppError.
func CodeOf(err error) Code {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return CodeServer
}

// ToProblemDetail convierte cualquier error a un ProblemDetail RFC 9457.
func ToProblemDetail(err error, instance string) model.ProblemDetail {
	code := CodeOf(err)
	return model.ProblemDetail{
		Type:     "/" + string(code),
		Title:    string(code),
		Status:   statusFor(code),
		Detail:   err.Error(),
		Instance: instance,
	}
}
