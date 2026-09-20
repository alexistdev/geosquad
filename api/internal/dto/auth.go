// Package dto memuat bentuk request dan response di batas HTTP. Sengaja
// dipisah dari model supaya perubahan kolom database tidak otomatis bocor
// jadi perubahan kontrak API.
package dto

import (
	"strings"

	"github.com/alexistdev/geosquad/api/internal/model"
)

type RegisterRequest struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterRequest) Normalize() {
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

func (r RegisterRequest) Validate() []string {
	var errs []string
	if r.FullName == "" {
		errs = append(errs, "Nama lengkap wajib diisi.")
	} else if len(r.FullName) > 150 {
		errs = append(errs, "Nama lengkap maksimal 150 karakter.")
	}
	errs = append(errs, validateEmail(r.Email)...)
	errs = append(errs, validatePassword(r.Password)...)
	return errs
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

func (r LoginRequest) Validate() []string {
	var errs []string
	if r.Email == "" {
		errs = append(errs, "Email wajib diisi.")
	}
	if r.Password == "" {
		errs = append(errs, "Password wajib diisi.")
	}
	return errs
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (r ChangePasswordRequest) Validate() []string {
	var errs []string
	if r.CurrentPassword == "" {
		errs = append(errs, "Password saat ini wajib diisi.")
	}
	errs = append(errs, validatePassword(r.NewPassword)...)
	if r.CurrentPassword != "" && r.CurrentPassword == r.NewPassword {
		errs = append(errs, "Password baru harus berbeda dari password saat ini.")
	}
	return errs
}

// UserResponse adalah bentuk user yang aman dikirim keluar.
type UserResponse struct {
	ID          string `json:"id"`
	FullName    string `json:"fullName"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	IsSuspended bool   `json:"isSuspended"`
	CreatedDate string `json:"createdDate"`
}

func NewUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		FullName:    u.FullName,
		Email:       u.Email,
		Role:        string(u.Role),
		IsSuspended: u.IsSuspended,
		CreatedDate: u.CreatedDate.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// LoginResponse mengikuti pola project lain: yang dikirim ke client adalah
// sessionId, bukan JWT-nya. JWT tinggal di Redis dan tidak pernah keluar server.
type LoginResponse struct {
	SessionID      string       `json:"sessionId"`
	User           UserResponse `json:"user"`
	DefaultHomeURL string       `json:"defaultHomeUrl"`
	ExpiresIn      int64        `json:"expiresIn"`
}
