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

type RunRepository struct {
	db *sql.DB
}

func NewRunRepository(db *sql.DB) *RunRepository {
	return &RunRepository{db: db}
}

// Nomor tiket dirakit di SELECT dari ticket_seq. Tidak disimpan sebagai
// kolom tersendiri supaya tidak ada dua nilai yang bisa berbeda pendapat.
const runColumns = `id, CONCAT('geo-', ticket_seq) AS ticket, user_id, request, status, branch, exit_code, error_message,
	started_at, finished_at, created_by, modified_by, created_date, modified_date, is_deleted`

func scanRun(row interface{ Scan(...any) error }) (*model.Run, error) {
	var r model.Run
	err := row.Scan(
		&r.ID, &r.Ticket, &r.UserID, &r.Request, &r.Status, &r.Branch, &r.ExitCode, &r.ErrorMessage,
		&r.StartedAt, &r.FinishedAt,
		&r.CreatedBy, &r.ModifiedBy, &r.CreatedDate, &r.ModifiedDate, &r.IsDeleted,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *RunRepository) Create(ctx context.Context, run *model.Run) error {
	const q = `INSERT INTO runs
		(id, user_id, request, status, created_by, modified_by, created_date, modified_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	// Sama seperti user: waktu diisi dari Go supaya struct yang dikembalikan
	// membawa nilai yang benar tanpa perlu membaca ulang barisnya.
	// Dipotong ke detik karena kolomnya DATETIME tanpa pecahan detik: MySQL
	// akan MEMBULATKAN .5 ke atas, sehingga nilai di struct dan di baris
	// database bisa berbeda satu detik.
	now := time.Now().UTC().Truncate(time.Second)
	res, err := r.db.ExecContext(ctx, q,
		run.ID, run.UserID, run.Request, run.Status, run.CreatedBy, run.ModifiedBy, now, now)
	if err != nil {
		return fmt.Errorf("simpan run: %w", err)
	}

	// Nomor tiket berasal dari AUTO_INCREMENT, jadi baru diketahui setelah
	// baris tersimpan. Dibaca balik di sini karena pemanggil membutuhkannya
	// segera: nilai itu diteruskan ke engine sebagai nama folder hasil kerja.
	seq, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("baca nomor tiket run: %w", err)
	}
	run.Ticket = fmt.Sprintf("geo-%d", seq)

	run.CreatedDate = now
	run.ModifiedDate = now
	return nil
}

func (r *RunRepository) FindByID(ctx context.Context, id string) (*model.Run, error) {
	q := `SELECT ` + runColumns + ` FROM runs WHERE id = ? AND is_deleted = 0`
	run, err := scanRun(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, fmt.Errorf("cari run: %w", err)
	}
	return run, nil
}

type RunFilter struct {
	// UserID kosong berarti lintas user. Hanya admin yang boleh begitu, dan
	// itu ditegakkan di service, bukan di sini.
	UserID string
	Status string
	Limit  int
	Offset int
}

func (r *RunRepository) List(ctx context.Context, f RunFilter) ([]model.Run, int64, error) {
	where := []string{"is_deleted = 0"}
	args := []any{}

	if f.UserID != "" {
		where = append(where, "user_id = ?")
		args = append(args, f.UserID)
	}
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	clause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung run: %w", err)
	}

	q := `SELECT ` + runColumns + ` FROM runs WHERE ` + clause +
		` ORDER BY created_date DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list run: %w", err)
	}
	defer rows.Close()

	runs := make([]model.Run, 0, f.Limit)
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("baca baris run: %w", err)
		}
		runs = append(runs, *run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi run: %w", err)
	}
	return runs, total, nil
}

func (r *RunRepository) MarkRunning(ctx context.Context, id, branch string) error {
	const q = `UPDATE runs SET status = ?, started_at = ?, branch = ?
		WHERE id = ? AND is_deleted = 0`
	res, err := r.db.ExecContext(ctx, q, model.RunRunning, time.Now().UTC(), nullIfEmpty(branch), id)
	if err != nil {
		return fmt.Errorf("tandai run berjalan: %w", err)
	}
	return requireAffected(res, "run")
}

func (r *RunRepository) SetBranch(ctx context.Context, id, branch string) error {
	const q = `UPDATE runs SET branch = ? WHERE id = ? AND is_deleted = 0`
	if _, err := r.db.ExecContext(ctx, q, branch, id); err != nil {
		return fmt.Errorf("simpan branch run: %w", err)
	}
	return nil
}

func (r *RunRepository) Finish(ctx context.Context, id string, status model.RunStatus, exitCode int, errMsg string) error {
	const q = `UPDATE runs
		SET status = ?, exit_code = ?, error_message = ?, finished_at = ?
		WHERE id = ? AND is_deleted = 0`

	res, err := r.db.ExecContext(ctx, q, status, exitCode, nullIfEmpty(errMsg), time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("selesaikan run: %w", err)
	}
	return requireAffected(res, "run")
}

// ResetStale mengembalikan run yang tertinggal menggantung saat proses API
// mati mendadak. Dipanggil sekali saat start: proses engine-nya sudah ikut
// mati bersama API, jadi status RUNNING di database pasti bohong.
func (r *RunRepository) ResetStale(ctx context.Context) (int64, error) {
	const q = `UPDATE runs
		SET status = ?, error_message = ?, finished_at = ?
		WHERE status IN (?, ?) AND is_deleted = 0`

	res, err := r.db.ExecContext(ctx, q,
		model.RunError, "Run terputus karena API restart.", time.Now().UTC(),
		model.RunPending, model.RunRunning)
	if err != nil {
		return 0, fmt.Errorf("reset run menggantung: %w", err)
	}
	return res.RowsAffected()
}

func (r *RunRepository) SoftDelete(ctx context.Context, id, modifiedBy string) error {
	const q = `UPDATE runs SET is_deleted = 1, modified_by = ? WHERE id = ? AND is_deleted = 0`
	res, err := r.db.ExecContext(ctx, q, modifiedBy, id)
	if err != nil {
		return fmt.Errorf("hapus run: %w", err)
	}
	return requireAffected(res, "run")
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
