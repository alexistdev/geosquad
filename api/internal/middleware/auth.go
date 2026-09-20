// Package middleware memuat lapisan yang membungkus setiap request.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/alexistdev/geosquad/api/internal/auth"
	"github.com/alexistdev/geosquad/api/internal/model"
	"github.com/alexistdev/geosquad/api/internal/repository"
	"github.com/alexistdev/geosquad/api/internal/response"
	"github.com/alexistdev/geosquad/api/internal/service"
)

type ctxKey string

const actorKey ctxKey = "actor"

// ActorFrom mengambil identitas pemanggil dari context. ok bernilai false
// kalau request belum melewati middleware Authenticate.
func ActorFrom(ctx context.Context) (*service.Actor, bool) {
	actor, ok := ctx.Value(actorKey).(*service.Actor)
	return actor, ok
}

type Auth struct {
	tokens     *auth.TokenService
	sessions   *auth.SessionStore
	users      *repository.UserRepository
	cookieName string
}

func NewAuth(tokens *auth.TokenService, sessions *auth.SessionStore, users *repository.UserRepository, cookieName string) *Auth {
	return &Auth{tokens: tokens, sessions: sessions, users: users, cookieName: cookieName}
}

// Authenticate menolak request yang tidak membawa sesi sah.
//
// Urutan resolusi mengikuti pola project lain: cookie SID lebih dulu, baru
// header Authorization. Cookie dipakai browser; header Bearer disediakan untuk
// pemanggil mesin yang memegang JWT langsung.
func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, sessionID, err := a.resolveToken(r)
		if err != nil {
			response.Unauthorized(w, "Sesi tidak ditemukan atau sudah kedaluwarsa. Silakan login lagi.")
			return
		}

		claims, err := a.tokens.Parse(token)
		if err != nil {
			response.Unauthorized(w, "Sesi tidak valid. Silakan login lagi.")
			return
		}

		// Peran dan status suspend dibaca ulang dari database, tidak dipercaya
		// dari klaim JWT. Tanpa ini, user yang baru saja diturunkan dari admin
		// tetap memegang hak admin sampai tokennya kedaluwarsa.
		user, err := a.users.FindByID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, repository.ErrNoRows) {
				response.Unauthorized(w, "Akun tidak ditemukan.")
				return
			}
			response.Internal(w, err)
			return
		}
		if user.IsSuspended {
			response.Forbidden(w, "Akun ini sedang dinonaktifkan.")
			return
		}

		actor := &service.Actor{
			UserID:    user.ID,
			Email:     user.Email,
			Role:      user.Role,
			SessionID: sessionID,
		}
		ctx := context.WithValue(r.Context(), actorKey, actor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) resolveToken(r *http.Request) (token, sessionID string, err error) {
	if c, cerr := r.Cookie(a.cookieName); cerr == nil && c.Value != "" {
		tok, rerr := a.sessions.Resolve(r.Context(), c.Value)
		if rerr == nil {
			return tok, c.Value, nil
		}
		// Cookie ada tapi sesinya sudah hilang dari Redis. Tidak jatuh ke
		// header: cookie basi harus terasa sebagai logout, bukan diam-diam
		// tergantikan kredensial lain.
		return "", "", rerr
	}

	// Sebagian klien tidak bisa memegang cookie. Untuk mereka, sessionId
	// boleh dikirim lewat header khusus.
	if sid := r.Header.Get("X-Session-Id"); sid != "" {
		tok, rerr := a.sessions.Resolve(r.Context(), sid)
		if rerr != nil {
			return "", "", rerr
		}
		return tok, sid, nil
	}

	if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authz, "Bearer ")), "", nil
	}

	return "", "", auth.ErrSessionNotFound
}

// RequireRole membatasi handler ke peran tertentu. Dipasang setelah
// Authenticate, dan akan menolak request kalau dipasang sebelumnya.
func RequireRole(roles ...model.Role) func(http.Handler) http.Handler {
	allowed := make(map[model.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := ActorFrom(r.Context())
			if !ok {
				response.Unauthorized(w, "Belum terautentikasi.")
				return
			}
			if _, permitted := allowed[actor.Role]; !permitted {
				response.Forbidden(w, "Akses ditolak untuk peran %s.", string(actor.Role))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
