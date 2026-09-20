// Package service memuat aturan bisnis. Tidak ada net/http di sini: service
// melempar error domain, handler yang menerjemahkannya jadi status HTTP.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/alexistdev/geosquad/api/internal/auth"
	"github.com/alexistdev/geosquad/api/internal/dto"
	"github.com/alexistdev/geosquad/api/internal/model"
	"github.com/alexistdev/geosquad/api/internal/repository"
	"github.com/alexistdev/geosquad/api/internal/response"
)

type AuthService struct {
	users    *repository.UserRepository
	tokens   *auth.TokenService
	sessions *auth.SessionStore
}

func NewAuthService(users *repository.UserRepository, tokens *auth.TokenService, sessions *auth.SessionStore) *AuthService {
	return &AuthService{users: users, tokens: tokens, sessions: sessions}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*model.User, error) {
	exists, err := s.users.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, response.Fail(response.ErrDuplicate, "Email %s sudah terdaftar.", req.Email)
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:       uuid.NewString(),
		FullName: req.FullName,
		Email:    req.Email,
		Password: hashed,
		// Registrasi mandiri selalu menghasilkan USER. Peran ADMIN hanya bisa
		// diberikan admin lain lewat endpoint manajemen user.
		Role: model.RoleUser,
		Audit: model.Audit{
			CreatedBy:  "Self-Register",
			ModifiedBy: "Self-Register",
		},
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	slog.Info("user terdaftar", "userId", user.ID, "email", user.Email)
	return user, nil
}

// Login memverifikasi kredensial, menerbitkan JWT, lalu menyimpannya di Redis
// di bawah sessionId acak. Yang dikembalikan ke pemanggil adalah sessionId.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			// Pesannya sengaja sama dengan kasus password salah. Membedakan
			// keduanya sama saja memberi tahu penyerang email mana yang
			// terdaftar.
			return nil, response.Fail(response.ErrUnauthorized, "Email atau password salah.")
		}
		return nil, err
	}

	if !auth.CheckPassword(user.Password, req.Password) {
		return nil, response.Fail(response.ErrUnauthorized, "Email atau password salah.")
	}
	if user.IsSuspended {
		return nil, response.Fail(response.ErrForbidden, "Akun ini sedang dinonaktifkan. Hubungi administrator.")
	}

	token, err := s.tokens.Generate(user)
	if err != nil {
		return nil, err
	}
	sessionID, err := s.sessions.Create(ctx, user.ID, token)
	if err != nil {
		return nil, err
	}

	slog.Info("login berhasil", "userId", user.ID, "role", user.Role)
	return &dto.LoginResponse{
		SessionID:      sessionID,
		User:           dto.NewUserResponse(user),
		DefaultHomeURL: defaultHomeURL(user.Role),
		ExpiresIn:      s.tokens.ExpiresIn(),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID, sessionID string) error {
	return s.sessions.Destroy(ctx, userID, sessionID)
}

func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	return s.sessions.DestroyAll(ctx, userID)
}

func (s *AuthService) Me(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return nil, response.Fail(response.ErrNotFound, "User tidak ditemukan.")
		}
		return nil, err
	}
	return user, nil
}

// ChangePassword mengganti password lalu mencabut semua sesi. Kalau password
// diganti karena dicurigai bocor, sesi lama harus ikut mati -- termasuk sesi
// yang sedang dipakai penyerang.
func (s *AuthService) ChangePassword(ctx context.Context, userID string, req dto.ChangePasswordRequest) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return response.Fail(response.ErrNotFound, "User tidak ditemukan.")
		}
		return err
	}

	if !auth.CheckPassword(user.Password, req.CurrentPassword) {
		return response.Fail(response.ErrUnauthorized, "Password saat ini salah.")
	}

	hashed, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, userID, hashed, user.Email); err != nil {
		return err
	}

	if err := s.sessions.DestroyAll(ctx, userID); err != nil {
		// Password sudah tersimpan. Gagal mencabut sesi itu serius, tapi
		// bukan alasan melaporkan penggantian password sebagai gagal.
		slog.Error("password berganti tapi sesi lama gagal dicabut", "userId", userID, "error", err)
	}
	slog.Info("password diganti", "userId", userID)
	return nil
}

func defaultHomeURL(role model.Role) string {
	return fmt.Sprintf("/%s/dashboard", strings.ToLower(string(role)))
}
