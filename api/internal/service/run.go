package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/alexistdev/geosquad/api/internal/dto"
	"github.com/alexistdev/geosquad/api/internal/model"
	"github.com/alexistdev/geosquad/api/internal/repository"
	"github.com/alexistdev/geosquad/api/internal/response"
	"github.com/alexistdev/geosquad/api/internal/runner"
)

// Engine adalah bagian runner yang dipakai service. Dibuat sebagai interface
// supaya service bisa diuji tanpa benar-benar menjalankan Python.
type Engine interface {
	Start(runID, ticket, request string)
	Cancel(runID string) bool
	IsRunning(runID string) bool
}

type RunService struct {
	runs   *repository.RunRepository
	logs   *repository.RunLogRepository
	engine Engine
	hub    *Hub

	// buffer menahan log sebentar sebelum ditulis batch ke database.
	mu      sync.Mutex
	buffers map[string]*logBuffer
}

type logBuffer struct {
	seq     int
	pending []model.RunLog
	timer   *time.Timer
}

const (
	// Log ditulis kalau sudah terkumpul sebanyak ini,
	flushSize = 50
	// atau kalau sudah sediam ini, mana yang lebih dulu. Tanpa batas waktu,
	// run yang lambat bicara akan menahan log terakhirnya di memori.
	flushInterval = 500 * time.Millisecond
)

func NewRunService(runs *repository.RunRepository, logs *repository.RunLogRepository, hub *Hub) *RunService {
	return &RunService{
		runs:    runs,
		logs:    logs,
		hub:     hub,
		buffers: make(map[string]*logBuffer),
	}
}

// AttachEngine menutup lingkaran ketergantungan: runner butuh service sebagai
// Sink, service butuh runner untuk menjalankan proses.
func (s *RunService) AttachEngine(e Engine) { s.engine = e }

var _ runner.Sink = (*RunService)(nil)

func (s *RunService) Create(ctx context.Context, req dto.CreateRunRequest, actor *Actor) (*model.Run, error) {
	run := &model.Run{
		ID:      uuid.NewString(),
		UserID:  actor.UserID,
		Request: req.Request,
		Status:  model.RunPending,
		Audit:   model.Audit{CreatedBy: actor.Email, ModifiedBy: actor.Email},
	}

	if err := s.runs.Create(ctx, run); err != nil {
		return nil, err
	}

	// Ditandai RUNNING di sini, bukan di dalam runner, supaya user yang baru
	// saja POST langsung melihat statusnya berubah tanpa menunggu slot.
	if err := s.runs.MarkRunning(ctx, run.ID, ""); err != nil {
		slog.Error("gagal menandai run berjalan", "runId", run.ID, "error", err)
	}
	run.Status = model.RunRunning

	s.engine.Start(run.ID, run.Ticket, run.Request)
	slog.Info("run dibuat", "runId", run.ID, "tiket", run.Ticket, "userId", actor.UserID)
	return run, nil
}

func (s *RunService) List(ctx context.Context, f repository.RunFilter, actor *Actor) ([]model.Run, int64, error) {
	// User biasa dipaksa hanya melihat miliknya, apa pun yang dia kirim di query.
	if !actor.IsAdmin() {
		f.UserID = actor.UserID
	}
	return s.runs.List(ctx, f)
}

func (s *RunService) Get(ctx context.Context, id string, actor *Actor) (*model.Run, error) {
	run, err := s.runs.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNoRows) {
			return nil, response.Fail(response.ErrNotFound, "Run tidak ditemukan.")
		}
		return nil, err
	}
	if !actor.CanAccess(run.UserID) {
		// Sengaja 404, bukan 403. Membalas 403 akan mengonfirmasi bahwa run
		// dengan id itu memang ada milik orang lain.
		return nil, response.Fail(response.ErrNotFound, "Run tidak ditemukan.")
	}
	return run, nil
}

func (s *RunService) Logs(ctx context.Context, runID string, afterSeq, limit int, actor *Actor) ([]model.RunLog, error) {
	if _, err := s.Get(ctx, runID, actor); err != nil {
		return nil, err
	}
	return s.logs.ListByRun(ctx, runID, afterSeq, limit)
}

func (s *RunService) Cancel(ctx context.Context, id string, actor *Actor) error {
	run, err := s.Get(ctx, id, actor)
	if err != nil {
		return err
	}
	if run.Status.Terminal() {
		return response.Fail(response.ErrConflict, "Run sudah selesai dengan status %s.", run.Status)
	}
	if !s.engine.Cancel(id) {
		return response.Fail(response.ErrConflict, "Run tidak sedang berjalan di server ini.")
	}
	slog.Info("run dibatalkan", "runId", id, "oleh", actor.Email)
	return nil
}

func (s *RunService) Delete(ctx context.Context, id string, actor *Actor) error {
	run, err := s.Get(ctx, id, actor)
	if err != nil {
		return err
	}
	if !run.Status.Terminal() {
		return response.Fail(response.ErrConflict, "Batalkan run dulu sebelum menghapusnya.")
	}
	return s.runs.SoftDelete(ctx, id, actor.Email)
}

// ResetStale dipanggil sekali saat start.
func (s *RunService) ResetStale(ctx context.Context) {
	n, err := s.runs.ResetStale(ctx)
	if err != nil {
		slog.Error("gagal mereset run menggantung", "error", err)
		return
	}
	if n > 0 {
		slog.Warn("run menggantung direset", "jumlah", n)
	}
}

// ---- implementasi runner.Sink ----------------------------------------------

func (s *RunService) OnLine(runID string, stream model.LogStream, line string) {
	s.mu.Lock()
	buf, ok := s.buffers[runID]
	if !ok {
		buf = &logBuffer{}
		s.buffers[runID] = buf
	}
	buf.seq++
	entry := model.RunLog{
		RunID:    runID,
		Seq:      buf.seq,
		Stream:   stream,
		Line:     truncate(line, 60000),
		LoggedAt: time.Now().UTC(),
	}
	buf.pending = append(buf.pending, entry)

	shouldFlush := len(buf.pending) >= flushSize
	if !shouldFlush {
		if buf.timer != nil {
			buf.timer.Stop()
		}
		buf.timer = time.AfterFunc(flushInterval, func() { s.flush(runID) })
	}
	s.mu.Unlock()

	// Siaran ke pelanggan SSE tidak menunggu database. Penonton log ingin
	// melihatnya sekarang, bukan setengah detik lagi.
	s.hub.Publish(runID, entry)

	if shouldFlush {
		s.flush(runID)
	}
}

func (s *RunService) OnBranch(runID, branch string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.runs.SetBranch(ctx, runID, branch); err != nil {
		slog.Error("gagal menyimpan branch run", "runId", runID, "error", err)
	}
}

func (s *RunService) OnFinish(runID string, status model.RunStatus, exitCode int, errMsg string) {
	s.flush(runID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.runs.Finish(ctx, runID, status, exitCode, errMsg); err != nil {
		slog.Error("gagal menyimpan hasil run", "runId", runID, "error", err)
	}

	s.mu.Lock()
	if buf, ok := s.buffers[runID]; ok && buf.timer != nil {
		buf.timer.Stop()
	}
	delete(s.buffers, runID)
	s.mu.Unlock()

	s.hub.Close(runID, status)
}

func (s *RunService) flush(runID string) {
	s.mu.Lock()
	buf, ok := s.buffers[runID]
	if !ok || len(buf.pending) == 0 {
		s.mu.Unlock()
		return
	}
	batch := buf.pending
	buf.pending = nil
	if buf.timer != nil {
		buf.timer.Stop()
	}
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.logs.AppendBatch(ctx, batch); err != nil {
		// Log yang hilang tidak boleh menghentikan run. Yang bisa dilakukan
		// hanya mencatat bahwa ada yang hilang.
		slog.Error("gagal menyimpan batch log", "runId", runID, "jumlah", len(batch), "error", err)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[dipotong]"
}
