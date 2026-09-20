package response

import (
	"errors"
	"fmt"
	"net/http"
)

// Error domain. Service melempar ini, handler menerjemahkannya ke HTTP, supaya
// service tidak perlu tahu apa-apa soal HTTP.
var (
	ErrNotFound     = errors.New("data tidak ditemukan")
	ErrDuplicate    = errors.New("data sudah ada")
	ErrUnauthorized = errors.New("tidak terautentikasi")
	ErrForbidden    = errors.New("tidak punya akses")
	ErrValidation   = errors.New("data tidak valid")
	ErrConflict     = errors.New("kondisi tidak memungkinkan")
)

// AppError membungkus error domain dengan pesan yang layak dibaca user.
type AppError struct {
	Kind    error
	Message string
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Kind }

func Fail(kind error, format string, args ...any) *AppError {
	return &AppError{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

// FromError memetakan error domain ke status HTTP. Error yang tidak dikenali
// dianggap 500 dan isinya tidak diteruskan ke client.
func FromError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		Internal(w, err)
		return
	}

	switch {
	case errors.Is(appErr.Kind, ErrNotFound):
		NotFound(w, appErr.Message)
	case errors.Is(appErr.Kind, ErrDuplicate):
		Conflict(w, appErr.Message)
	case errors.Is(appErr.Kind, ErrConflict):
		Conflict(w, appErr.Message)
	case errors.Is(appErr.Kind, ErrUnauthorized):
		Unauthorized(w, appErr.Message)
	case errors.Is(appErr.Kind, ErrForbidden):
		Forbidden(w, appErr.Message)
	case errors.Is(appErr.Kind, ErrValidation):
		BadRequest(w, appErr.Message)
	default:
		Internal(w, err)
	}
}
