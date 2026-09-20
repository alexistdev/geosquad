package handler

import (
	"net/http"
	"time"

	"github.com/alexistdev/geosquad/api/internal/config"
	"github.com/alexistdev/geosquad/api/internal/dto"
	"github.com/alexistdev/geosquad/api/internal/response"
	"github.com/alexistdev/geosquad/api/internal/service"
)

type AuthHandler struct {
	svc    *service.AuthService
	cookie config.JWTConfig
}

func NewAuthHandler(svc *service.AuthService, cookie config.JWTConfig) *AuthHandler {
	return &AuthHandler{svc: svc, cookie: cookie}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if !decode(w, r, &req) {
		return
	}
	req.Normalize()
	if errs := req.Validate(); len(errs) > 0 {
		response.BadRequest(w, errs...)
		return
	}

	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.Created(w, "Registrasi berhasil.", dto.NewUserResponse(user))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if !decode(w, r, &req) {
		return
	}
	req.Normalize()
	if errs := req.Validate(); len(errs) > 0 {
		response.BadRequest(w, errs...)
		return
	}

	result, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.FromError(w, err)
		return
	}

	h.setSessionCookie(w, result.SessionID)
	response.OK(w, "Login berhasil.", result)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	if a.SessionID != "" {
		if err := h.svc.Logout(r.Context(), a.UserID, a.SessionID); err != nil {
			response.FromError(w, err)
			return
		}
	}
	h.clearSessionCookie(w)
	response.NoContent(w, "Logout berhasil.")
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}
	if err := h.svc.LogoutAll(r.Context(), a.UserID); err != nil {
		response.FromError(w, err)
		return
	}
	h.clearSessionCookie(w)
	response.NoContent(w, "Semua sesi telah dikeluarkan.")
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}
	user, err := h.svc.Me(r.Context(), a.UserID)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "", dto.NewUserResponse(user))
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	var req dto.ChangePasswordRequest
	if !decode(w, r, &req) {
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		response.BadRequest(w, errs...)
		return
	}

	if err := h.svc.ChangePassword(r.Context(), a.UserID, req); err != nil {
		response.FromError(w, err)
		return
	}
	// Semua sesi dicabut, termasuk yang sedang dipakai. Cookie-nya ikut
	// dibersihkan supaya browser tidak menyimpan sesi yang sudah mati.
	h.clearSessionCookie(w)
	response.NoContent(w, "Password berhasil diganti. Silakan login lagi.")
}

// setSessionCookie menaruh sessionId di cookie HttpOnly.
//
// HttpOnly berarti JavaScript tidak bisa membacanya, sehingga XSS tidak
// otomatis berarti sesi tercuri. SameSite=Lax menahan CSRF untuk request
// lintas situs yang mengubah data, sambil tetap mengizinkan navigasi biasa.
func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.CookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookie.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.cookie.SessionTTL.Seconds()),
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookie.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
