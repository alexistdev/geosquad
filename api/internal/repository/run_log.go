package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alexistdev/geosquad/api/internal/model"
)

type RunLogRepository struct {
	db *sql.DB
}

func NewRunLogRepository(db *sql.DB) *RunLogRepository {
	return &RunLogRepository{db: db}
}

// AppendBatch menulis beberapa baris log sekaligus.
//
// Sengaja batch, bukan satu INSERT per baris: engine yang verbose bisa
// mengeluarkan ratusan baris per detik, dan satu round-trip per baris akan
// membuat database jadi penghambat proses yang seharusnya cuma diamati.
func (r *RunLogRepository) AppendBatch(ctx context.Context, logs []model.RunLog) error {
	if len(logs) == 0 {
		return nil
	}

	query := `INSERT INTO run_logs (run_id, seq, stream, line) VALUES `
	args := make([]any, 0, len(logs)*4)
	for i, l := range logs {
		if i > 0 {
			query += ","
		}
		query += "(?, ?, ?, ?)"
		args = append(args, l.RunID, l.Seq, l.Stream, l.Line)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("simpan log run: %w", err)
	}
	return nil
}

// ListByRun mengembalikan log setelah seq tertentu. afterSeq = 0 berarti dari
// awal. Dipakai juga oleh SSE untuk mengirim riwayat sebelum menyusul live.
func (r *RunLogRepository) ListByRun(ctx context.Context, runID string, afterSeq, limit int) ([]model.RunLog, error) {
	const q = `SELECT id, run_id, seq, stream, line, logged_at
		FROM run_logs
		WHERE run_id = ? AND seq > ?
		ORDER BY seq ASC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, q, runID, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("baca log run: %w", err)
	}
	defer rows.Close()

	logs := make([]model.RunLog, 0, limit)
	for rows.Next() {
		var l model.RunLog
		if err := rows.Scan(&l.ID, &l.RunID, &l.Seq, &l.Stream, &l.Line, &l.LoggedAt); err != nil {
			return nil, fmt.Errorf("baca baris log: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi log: %w", err)
	}
	return logs, nil
}

func (r *RunLogRepository) LastSeq(ctx context.Context, runID string) (int, error) {
	const q = `SELECT COALESCE(MAX(seq), 0) FROM run_logs WHERE run_id = ?`
	var seq int
	if err := r.db.QueryRowContext(ctx, q, runID).Scan(&seq); err != nil {
		return 0, fmt.Errorf("baca seq terakhir: %w", err)
	}
	return seq, nil
}
