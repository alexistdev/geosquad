package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/alexistdev/geosquad/api/internal/response"
	"github.com/google/uuid"
)

// statusWriter mengintip status code yang ditulis handler, karena
// http.ResponseWriter sendiri tidak menyediakannya.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Flush meneruskan ke ResponseWriter aslinya. Tanpa ini, SSE berhenti bekerja
// begitu handler-nya dibungkus middleware logging.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		level := slog.LevelInfo
		if sw.status >= 500 {
			level = slog.LevelError
		} else if sw.status >= 400 {
			level = slog.LevelWarn
		}

		slog.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"durasi", time.Since(start).Round(time.Millisecond).String(),
			"requestId", w.Header().Get("X-Request-Id"),
		)
	})
}

// RequestID memberi tiap request penanda, supaya satu keluhan user bisa
// ditelusuri di log tanpa menebak-nebak.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}

// Recover mencegah satu panic menjatuhkan seluruh server.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic pada handler",
					"error", rec,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				response.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS hanya memantulkan origin yang memang terdaftar.
//
// Wildcard "*" sengaja tidak didukung: API ini memakai cookie sesi, dan
// browser menolak kombinasi Allow-Origin "*" dengan Allow-Credentials true.
func CORS(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && slices.Contains(allowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
					"Content-Type", "Authorization", "X-Session-Id", "X-Request-Id",
				}, ", "))
				w.Header().Set("Access-Control-Max-Age", "600")
				// Response bisa berbeda per origin, jadi cache harus tahu.
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders memasang header pertahanan dasar.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
