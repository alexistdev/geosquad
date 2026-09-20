// Package response menyeragamkan bentuk seluruh response API.
//
// Bentuknya sengaja sama dengan project lain (geobill, geolicense):
// {status, messages, payload}. Frontend jadi bisa memakai interceptor yang
// sama tanpa cabang per-project.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Data struct {
	Status   bool     `json:"status"`
	Messages []string `json:"messages"`
	Payload  any      `json:"payload,omitempty"`
}

// Meta dipakai endpoint yang berhalaman.
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"perPage"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type Paged struct {
	Items any  `json:"items"`
	Meta  Meta `json:"meta"`
}

func write(w http.ResponseWriter, code int, body Data) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Header sudah terkirim, jadi tidak bisa diubah jadi 500. Yang bisa
		// dilakukan hanya mencatatnya.
		slog.Error("gagal menulis response", "error", err)
	}
}

func OK(w http.ResponseWriter, message string, payload any) {
	write(w, http.StatusOK, Data{Status: true, Messages: msg(message), Payload: payload})
}

func Created(w http.ResponseWriter, message string, payload any) {
	write(w, http.StatusCreated, Data{Status: true, Messages: msg(message), Payload: payload})
}

func NoContent(w http.ResponseWriter, message string) {
	write(w, http.StatusOK, Data{Status: true, Messages: msg(message)})
}

func Error(w http.ResponseWriter, code int, messages ...string) {
	if len(messages) == 0 {
		messages = []string{http.StatusText(code)}
	}
	write(w, code, Data{Status: false, Messages: messages})
}

func BadRequest(w http.ResponseWriter, messages ...string) {
	Error(w, http.StatusBadRequest, messages...)
}

func Unauthorized(w http.ResponseWriter, messages ...string) {
	Error(w, http.StatusUnauthorized, messages...)
}

func Forbidden(w http.ResponseWriter, messages ...string) {
	Error(w, http.StatusForbidden, messages...)
}

func NotFound(w http.ResponseWriter, messages ...string) {
	Error(w, http.StatusNotFound, messages...)
}

func Conflict(w http.ResponseWriter, messages ...string) {
	Error(w, http.StatusConflict, messages...)
}

// Internal sengaja tidak pernah membocorkan isi error ke client. Detailnya
// masuk log, yang dilihat client cuma kalimat umum.
func Internal(w http.ResponseWriter, err error) {
	slog.Error("kesalahan internal", "error", err)
	Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
}

func msg(m string) []string {
	if m == "" {
		return []string{}
	}
	return []string{m}
}

// Unhealthy membalas 503 tapi tetap membawa payload diagnosa. Dipakai health
// check, di mana operator butuh tahu dependensi mana yang jatuh.
func Unhealthy(w http.ResponseWriter, message string, payload any) {
	write(w, http.StatusServiceUnavailable, Data{Status: false, Messages: msg(message), Payload: payload})
}
