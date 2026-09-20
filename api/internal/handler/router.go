package handler

import (
	"net/http"

	"github.com/alexistdev/geosquad/api/internal/config"
	"github.com/alexistdev/geosquad/api/internal/middleware"
	"github.com/alexistdev/geosquad/api/internal/model"
	"github.com/alexistdev/geosquad/api/internal/response"
)

type Handlers struct {
	Health *HealthHandler
	Auth   *AuthHandler
	User   *UserHandler
	Run    *RunHandler
}

// NewRouter merangkai seluruh route.
//
// Router-nya http.ServeMux bawaan. Sejak Go 1.22 ia sudah mengerti method dan
// path parameter ("POST /runs/{id}"), yang dulu jadi alasan utama memakai
// router pihak ketiga.
func NewRouter(cfg *config.Config, h Handlers, authMW *middleware.Auth) http.Handler {
	mux := http.NewServeMux()

	protected := func(fn http.HandlerFunc) http.Handler {
		return authMW.Authenticate(fn)
	}
	adminOnly := func(fn http.HandlerFunc) http.Handler {
		return middleware.Chain(fn,
			authMW.Authenticate,
			middleware.RequireRole(model.RoleAdmin),
		)
	}

	// ---- publik --------------------------------------------------------
	mux.HandleFunc("GET /health", h.Health.Health)
	mux.HandleFunc("POST /api/v1/auth/register", h.Auth.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Auth.Login)

	// ---- butuh login ---------------------------------------------------
	mux.Handle("POST /api/v1/auth/logout", protected(h.Auth.Logout))
	mux.Handle("POST /api/v1/auth/logout-all", protected(h.Auth.LogoutAll))
	mux.Handle("GET /api/v1/auth/me", protected(h.Auth.Me))
	mux.Handle("POST /api/v1/auth/change-password", protected(h.Auth.ChangePassword))

	mux.Handle("POST /api/v1/runs", protected(h.Run.Create))
	mux.Handle("GET /api/v1/runs", protected(h.Run.List))
	mux.Handle("GET /api/v1/runs/{id}", protected(h.Run.Get))
	mux.Handle("GET /api/v1/runs/{id}/logs", protected(h.Run.Logs))
	mux.Handle("GET /api/v1/runs/{id}/stream", protected(h.Run.Stream))
	mux.Handle("POST /api/v1/runs/{id}/cancel", protected(h.Run.Cancel))
	mux.Handle("DELETE /api/v1/runs/{id}", protected(h.Run.Delete))

	// ---- admin ---------------------------------------------------------
	mux.Handle("GET /api/v1/admin/users", adminOnly(h.User.List))
	mux.Handle("POST /api/v1/admin/users", adminOnly(h.User.Create))
	mux.Handle("GET /api/v1/admin/users/{id}", adminOnly(h.User.Get))
	mux.Handle("PATCH /api/v1/admin/users/{id}", adminOnly(h.User.Update))
	mux.Handle("DELETE /api/v1/admin/users/{id}", adminOnly(h.User.Delete))

	// ServeMux membalas 404 dengan teks biasa. Diganti supaya seluruh API
	// konsisten mengembalikan JSON, termasuk saat salah alamat.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response.NotFound(w, "Endpoint tidak ditemukan: "+r.Method+" "+r.URL.Path)
	})

	return middleware.Chain(mux,
		middleware.Recover,
		middleware.RequestID,
		middleware.RequestLogger,
		middleware.SecureHeaders,
		middleware.CORS(cfg.CORSAllowedOrigins),
	)
}
