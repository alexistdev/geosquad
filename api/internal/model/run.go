package model

import "time"

// RunStatus menggambarkan siklus hidup satu run engine.
//
//	PENDING   -> baru dibuat, belum dapat slot
//	RUNNING   -> proses engine sedang jalan
//	PASSED    -> engine selesai exit 0, SQA lulus
//	FAILED    -> engine selesai exit bukan 0, SQA masih menolak
//	CANCELLED -> dihentikan user
//	ERROR     -> gagal di luar logika engine (proses tidak bisa start, timeout)
type RunStatus string

const (
	RunPending   RunStatus = "PENDING"
	RunRunning   RunStatus = "RUNNING"
	RunPassed    RunStatus = "PASSED"
	RunFailed    RunStatus = "FAILED"
	RunCancelled RunStatus = "CANCELLED"
	RunError     RunStatus = "ERROR"
)

// Terminal menandai status yang tidak akan berubah lagi.
func (s RunStatus) Terminal() bool {
	switch s {
	case RunPassed, RunFailed, RunCancelled, RunError:
		return true
	}
	return false
}

type Run struct {
	ID string `json:"id"`
	// Ticket adalah nomor yang dilihat manusia (geo-1001) dan sekaligus nama
	// folder hasil kerja di engine. Diisi database lewat kolom turunan, tidak
	// pernah oleh Go.
	Ticket       string     `json:"ticket"`
	UserID       string     `json:"userId"`
	Request      string     `json:"request"`
	Status       RunStatus  `json:"status"`
	Branch       *string    `json:"branch,omitempty"`
	ExitCode     *int       `json:"exitCode,omitempty"`
	ErrorMessage *string    `json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	Audit
}

type LogStream string

const (
	StreamStdout LogStream = "STDOUT"
	StreamStderr LogStream = "STDERR"
	StreamSystem LogStream = "SYSTEM"
)

type RunLog struct {
	ID       int64     `json:"id"`
	RunID    string    `json:"runId"`
	Seq      int       `json:"seq"`
	Stream   LogStream `json:"stream"`
	Line     string    `json:"line"`
	LoggedAt time.Time `json:"loggedAt"`
}
