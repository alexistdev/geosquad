package dto

import (
	"strings"
	"time"

	"github.com/alexistdev/geosquad/api/internal/model"
)

// CreateUserRequest dipakai admin untuk membuat user, termasuk menentukan peran.
type CreateUserRequest struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (r *CreateUserRequest) Normalize() {
	r.FullName = strings.TrimSpace(r.FullName)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Role = strings.ToUpper(strings.TrimSpace(r.Role))
	if r.Role == "" {
		r.Role = string(model.RoleUser)
	}
}

func (r CreateUserRequest) Validate() []string {
	var errs []string
	if r.FullName == "" {
		errs = append(errs, "Nama lengkap wajib diisi.")
	}
	errs = append(errs, validateEmail(r.Email)...)
	errs = append(errs, validatePassword(r.Password)...)
	if !model.Role(r.Role).Valid() {
		errs = append(errs, "Role harus ADMIN atau USER.")
	}
	return errs
}

// UpdateUserRequest memakai pointer supaya bisa membedakan "tidak dikirim"
// dari "dikirim kosong". Tanpa itu, PATCH akan menghapus field yang diam saja.
type UpdateUserRequest struct {
	FullName    *string `json:"fullName"`
	Role        *string `json:"role"`
	IsSuspended *bool   `json:"isSuspended"`
}

func (r *UpdateUserRequest) Normalize() {
	if r.FullName != nil {
		trimmed := strings.TrimSpace(*r.FullName)
		r.FullName = &trimmed
	}
	if r.Role != nil {
		upper := strings.ToUpper(strings.TrimSpace(*r.Role))
		r.Role = &upper
	}
}

func (r UpdateUserRequest) Validate() []string {
	var errs []string
	if r.FullName == nil && r.Role == nil && r.IsSuspended == nil {
		errs = append(errs, "Tidak ada field yang diubah.")
	}
	if r.FullName != nil && *r.FullName == "" {
		errs = append(errs, "Nama lengkap tidak boleh kosong.")
	}
	if r.Role != nil && !model.Role(*r.Role).Valid() {
		errs = append(errs, "Role harus ADMIN atau USER.")
	}
	return errs
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format("2006-01-02T15:04:05Z")
	return &s
}
