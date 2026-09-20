// Package repository memegang semua SQL. Tidak ada query di luar package ini.
//
// Dua aturan yang berlaku di sini:
//  1. Semua query menyaring is_deleted = 0. Hapus selalu soft delete.
//  2. Parameter selalu lewat placeholder, tidak pernah dirangkai dengan fmt.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alexistdev/geosquad/api/internal/model"
)

var ErrNoRows = sql.ErrNoRows

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, full_name, email, password, role, is_suspended,
	created_by, modified_by, created_date, modified_date, is_deleted`

func scanUser(row interface{ Scan(...any) error }) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.FullName, &u.Email, &u.Password, &u.Role, &u.IsSuspended,
		&u.CreatedBy, &u.ModifiedBy, &u.CreatedDate, &u.ModifiedDate, &u.IsDeleted,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	const q = `INSERT INTO users
		(id, full_name, email, password, role, is_suspended, created_by, modified_by,
		 created_date, modified_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Waktu ditentukan di sini, bukan dibiarkan ke DEFAULT CURRENT_TIMESTAMP,
	// supaya struct yang dikembalikan ke pemanggil membawa nilai yang sama
	// dengan yang tersimpan. Tanpa ini response menampilkan tanggal nol.
	// Dipotong ke detik karena kolomnya DATETIME tanpa pecahan detik: MySQL
	// akan MEMBULATKAN .5 ke atas, sehingga nilai di struct dan di baris
	// database bisa berbeda satu detik.
	now := time.Now().UTC().Truncate(time.Second)
	_, err := r.db.ExecContext(ctx, q,
		u.ID, u.FullName, u.Email, u.Password, u.Role, u.IsSuspended, u.CreatedBy, u.ModifiedBy,
		now, now)
	if err != nil {
		return fmt.Errorf("simpan user: %w", err)
	}

	u.CreatedDate = now
	u.ModifiedDate = now
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE id = ? AND is_deleted = 0`
	u, err := scanUser(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, fmt.Errorf("cari user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	q := `SELECT ` + userColumns + ` FROM users WHERE email = ? AND is_deleted = 0`
	u, err := scanUser(r.db.QueryRowContext(ctx, q, email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, fmt.Errorf("cari user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const q = `SELECT 1 FROM users WHERE email = ? AND is_deleted = 0 LIMIT 1`
	var one int
	err := r.db.QueryRowContext(ctx, q, email).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cek email terpakai: %w", err)
	}
	return true, nil
}

type UserFilter struct {
	Search string
	Role   string
	Limit  int
	Offset int
}

func (r *UserRepository) List(ctx context.Context, f UserFilter) ([]model.User, int64, error) {
	where := []string{"is_deleted = 0"}
	args := []any{}

	if f.Search != "" {
		where = append(where, "(full_name LIKE ? OR email LIKE ?)")
		like := "%" + f.Search + "%"
		args = append(args, like, like)
	}
	if f.Role != "" {
		where = append(where, "role = ?")
		args = append(args, f.Role)
	}
	clause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung user: %w", err)
	}

	q := `SELECT ` + userColumns + ` FROM users WHERE ` + clause +
		` ORDER BY created_date DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list user: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0, f.Limit)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("baca baris user: %w", err)
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi user: %w", err)
	}
	return users, total, nil
}

func (r *UserRepository) Update(ctx context.Context, u *model.User) error {
	const q = `UPDATE users
		SET full_name = ?, role = ?, is_suspended = ?, modified_by = ?
		WHERE id = ? AND is_deleted = 0`

	res, err := r.db.ExecContext(ctx, q, u.FullName, u.Role, u.IsSuspended, u.ModifiedBy, u.ID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return requireAffected(res, "user")
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, hashed, modifiedBy string) error {
	const q = `UPDATE users SET password = ?, modified_by = ? WHERE id = ? AND is_deleted = 0`
	res, err := r.db.ExecContext(ctx, q, hashed, modifiedBy, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return requireAffected(res, "user")
}

// SoftDelete menandai baris terhapus. Email sengaja ikut diubah supaya alamat
// yang sama bisa dipakai mendaftar lagi tanpa menabrak unique index.
func (r *UserRepository) SoftDelete(ctx context.Context, id, modifiedBy string) error {
	const q = `UPDATE users
		SET is_deleted = 1,
		    email = CONCAT(email, '.deleted.', UNIX_TIMESTAMP()),
		    modified_by = ?
		WHERE id = ? AND is_deleted = 0`

	res, err := r.db.ExecContext(ctx, q, modifiedBy, id)
	if err != nil {
		return fmt.Errorf("hapus user: %w", err)
	}
	return requireAffected(res, "user")
}

// CountAdmins dipakai untuk mencegah admin terakhir menghapus atau
// menurunkan dirinya sendiri, yang akan mengunci semua orang keluar.
func (r *UserRepository) CountAdmins(ctx context.Context) (int64, error) {
	const q = `SELECT COUNT(*) FROM users WHERE role = 'ADMIN' AND is_deleted = 0 AND is_suspended = 0`
	var n int64
	if err := r.db.QueryRowContext(ctx, q).Scan(&n); err != nil {
		return 0, fmt.Errorf("hitung admin: %w", err)
	}
	return n, nil
}

func requireAffected(res sql.Result, entity string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("baca rows affected %s: %w", entity, err)
	}
	if n == 0 {
		return ErrNoRows
	}
	return nil
}
