package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/alexistdev/geosquad/api/internal/auth"
	"github.com/alexistdev/geosquad/api/internal/dto"
	"github.com/alexistdev/geosquad/api/internal/model"
	"github.com/alexistdev/geosquad/api/internal/repository"
	"github.com/alexistdev/geosquad/api/internal/response"
)

type UserService struct {
	users    *repository.UserRepository
	sessions *auth.SessionStore
}

func NewUserService(users *repository.UserRepository, sessions *auth.SessionStore) *UserService {
	return &UserService{users: users, sessions: sessions}
}

func (s *UserService) List(ctx context.Context, f repository.UserFilter) ([]model.User, int64, error) {
	return s.users.List(ctx, f)
}

func (s *UserService) Get(ctx context.Context, id string) (*model.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return nil, response.Fail(response.ErrNotFound, "User tidak ditemukan.")
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) Create(ctx context.Context, req dto.CreateUserRequest, actor string) (*model.User, error) {
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
		Role:     model.Role(req.Role),
		Audit:    model.Audit{CreatedBy: actor, ModifiedBy: actor},
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	slog.Info("user dibuat admin", "userId", user.ID, "role", user.Role, "oleh", actor)
	return user, nil
}

// Update mengubah nama, peran, atau status suspend.
//
// Dua pengaman di sini melindungi dari keadaan yang tidak bisa dipulihkan
// lewat API: admin aktif terakhir tidak boleh diturunkan atau dinonaktifkan,
// dan admin tidak boleh menonaktifkan dirinya sendiri.
func (s *UserService) Update(ctx context.Context, id string, req dto.UpdateUserRequest, actor *Actor) (*model.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return nil, response.Fail(response.ErrNotFound, "User tidak ditemukan.")
		}
		return nil, err
	}

	losingAdmin := user.Role.IsAdmin() &&
		((req.Role != nil && *req.Role != string(model.RoleAdmin)) ||
			(req.IsSuspended != nil && *req.IsSuspended && !user.IsSuspended))

	if losingAdmin {
		if user.ID == actor.UserID && req.IsSuspended != nil && *req.IsSuspended {
			return nil, response.Fail(response.ErrConflict, "Tidak bisa menonaktifkan akun sendiri.")
		}
		admins, err := s.users.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if admins <= 1 {
			return nil, response.Fail(response.ErrConflict,
				"Ini admin aktif terakhir. Angkat admin lain dulu sebelum mengubah yang ini.")
		}
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Role != nil {
		user.Role = model.Role(*req.Role)
	}
	suspending := req.IsSuspended != nil && *req.IsSuspended && !user.IsSuspended
	if req.IsSuspended != nil {
		user.IsSuspended = *req.IsSuspended
	}
	user.ModifiedBy = actor.Email

	if err := s.users.Update(ctx, user); err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return nil, response.Fail(response.ErrNotFound, "User tidak ditemukan.")
		}
		return nil, err
	}

	// Suspend baru benar-benar berlaku kalau sesi yang sedang jalan ikut
	// dicabut. Tanpa ini, user yang dinonaktifkan tetap bisa memakai API
	// sampai sesinya kedaluwarsa sendiri.
	if suspending {
		if err := s.sessions.DestroyAll(ctx, user.ID); err != nil {
			slog.Error("gagal mencabut sesi user yang disuspend", "userId", user.ID, "error", err)
		}
	}

	slog.Info("user diubah", "userId", user.ID, "oleh", actor.Email)
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id string, actor *Actor) error {
	if id == actor.UserID {
		return response.Fail(response.ErrConflict, "Tidak bisa menghapus akun sendiri.")
	}

	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return response.Fail(response.ErrNotFound, "User tidak ditemukan.")
		}
		return err
	}

	if user.Role.IsAdmin() {
		admins, err := s.users.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return response.Fail(response.ErrConflict, "Ini admin aktif terakhir dan tidak bisa dihapus.")
		}
	}

	if err := s.users.SoftDelete(ctx, id, actor.Email); err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return response.Fail(response.ErrNotFound, "User tidak ditemukan.")
		}
		return err
	}
	if err := s.sessions.DestroyAll(ctx, id); err != nil {
		slog.Error("gagal mencabut sesi user yang dihapus", "userId", id, "error", err)
	}

	slog.Info("user dihapus", "userId", id, "oleh", actor.Email)
	return nil
}
