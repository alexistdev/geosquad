// Package runner menjalankan engine sebagai proses terpisah dan mengalirkan
// keluarannya ke database serta ke pelanggan SSE.
//
// Tiga hal yang membentuk file ini:
//
//  1. API tidak pernah menunggu engine selesai di dalam request. Engine bisa
//     jalan puluhan menit; request HTTP tidak boleh menggantung selama itu.
//  2. Argumen ke engine hanya satu, yaitu teks request user, dan diberikan
//     sebagai argumen exec.Command — bukan lewat shell. Tanpa shell, tidak ada
//     yang perlu di-escape dan tidak ada celah command injection.
//  3. Jumlah proses dibatasi semaphore. Satu run engine memakai CPU dan kuota
//     LLM; tanpa batas, sepuluh request bersamaan akan saling mematikan.
package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/alexistdev/geosquad/api/internal/config"
	"github.com/alexistdev/geosquad/api/internal/model"
)

// Sink menerima kejadian dari proses engine. Implementasinya ada di service,
// supaya package ini tidak perlu tahu soal database.
type Sink interface {
	OnLine(runID string, stream model.LogStream, line string)
	OnBranch(runID, branch string)
	OnFinish(runID string, status model.RunStatus, exitCode int, errMsg string)
}

// Engine mencetak baris "[git] branch: squad/20260919-203145" di awal run.
// Dari situ branch hasil kerja bisa dicatat tanpa perlu membaca git workspace.
var branchPattern = regexp.MustCompile(`\[git\]\s+branch:\s+(\S+)`)

type Runner struct {
	cfg  config.EngineConfig
	sink Sink

	// slots membatasi jumlah proses engine yang boleh jalan bersamaan.
	slots chan struct{}

	mu      sync.Mutex
	running map[string]context.CancelFunc
}

func New(cfg config.EngineConfig, sink Sink) *Runner {
	return &Runner{
		cfg:     cfg,
		sink:    sink,
		slots:   make(chan struct{}, max(1, cfg.MaxConcurrent)),
		running: make(map[string]context.CancelFunc),
	}
}

// Validate memastikan engine benar-benar ada sebelum API menerima request.
// Lebih baik gagal saat start daripada baru ketahuan saat user menekan tombol.
func (r *Runner) Validate() error {
	main := filepath.Join(r.cfg.Dir, "main.py")
	if _, err := os.Stat(main); err != nil {
		return fmt.Errorf("engine tidak ditemukan di %s: %w", main, err)
	}
	if _, err := os.Stat(r.cfg.Python); err != nil {
		return fmt.Errorf("interpreter engine tidak ditemukan di %s (jalankan setup venv engine dulu): %w", r.cfg.Python, err)
	}
	return nil
}

// Start menjalankan satu run di goroutine terpisah dan langsung kembali.
//
// ticket diteruskan ke engine sebagai SQUAD_TICKET, dan engine memakainya
// sebagai nama folder workspace, output, serta venv. Dengan begitu hasil tiap
// task terpisah dan tidak lagi saling menimpa.
func (r *Runner) Start(runID, ticket, request string) {
	go r.execute(runID, ticket, request)
}

// Cancel menghentikan run yang sedang jalan. Mengembalikan false kalau run
// itu tidak sedang dipegang proses ini.
func (r *Runner) Cancel(runID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	cancel, ok := r.running[runID]
	if !ok {
		return false
	}
	cancel()
	return true
}

func (r *Runner) IsRunning(runID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.running[runID]
	return ok
}

func (r *Runner) execute(runID, ticket, request string) {
	// Menunggu slot. Selama antre, status di database tetap PENDING sehingga
	// user melihat run-nya diterima tapi belum jalan.
	r.slots <- struct{}{}
	defer func() { <-r.slots }()

	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.Timeout)
	defer cancel()

	r.mu.Lock()
	r.running[runID] = cancel
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.running, runID)
		r.mu.Unlock()
	}()

	absDir, err := filepath.Abs(r.cfg.Dir)
	if err != nil {
		r.sink.OnFinish(runID, model.RunError, -1, fmt.Sprintf("path engine tidak valid: %v", err))
		return
	}
	absPython, err := filepath.Abs(r.cfg.Python)
	if err != nil {
		r.sink.OnFinish(runID, model.RunError, -1, fmt.Sprintf("path interpreter tidak valid: %v", err))
		return
	}

	// Request masuk sebagai satu argumen utuh. Tidak lewat shell, jadi karakter
	// apa pun di dalamnya tidak punya arti khusus.
	cmd := exec.CommandContext(ctx, absPython, "main.py", request)
	cmd.Dir = absDir
	cmd.Env = append(os.Environ(),
		// Tanpa ini Python membufer stdout saat dipipe, dan log baru muncul
		// berombak di akhir alih-alih mengalir realtime.
		"PYTHONUNBUFFERED=1",
		// Engine memakai ini sebagai nama folder hasil kerja. Lewat env, bukan
		// argumen, supaya tidak tercampur dengan teks request user.
		"SQUAD_TICKET="+ticket,
	)
	// Engine menanyakan konfirmasi deploy lewat stdin. Diberi stdin kosong
	// supaya ask_approval melihat sesi non-interaktif dan melewati deploy,
	// bukan menggantung menunggu ketikan yang tidak akan pernah datang.
	cmd.Stdin = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.sink.OnFinish(runID, model.RunError, -1, fmt.Sprintf("gagal menyiapkan stdout: %v", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.sink.OnFinish(runID, model.RunError, -1, fmt.Sprintf("gagal menyiapkan stderr: %v", err))
		return
	}

	slog.Info("menjalankan engine", "runId", runID, "tiket", ticket, "dir", absDir)
	if err := cmd.Start(); err != nil {
		r.sink.OnFinish(runID, model.RunError, -1, fmt.Sprintf("gagal menjalankan engine: %v", err))
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); r.pump(runID, stdout, model.StreamStdout) }()
	go func() { defer wg.Done(); r.pump(runID, stderr, model.StreamStderr) }()
	// Tunggu kedua pipe habis sebelum Wait, supaya tidak ada baris log yang
	// hilang karena proses keburu dinyatakan selesai.
	wg.Wait()

	waitErr := cmd.Wait()
	status, exitCode, errMsg := classify(ctx, waitErr)

	slog.Info("engine selesai", "runId", runID, "status", status, "exitCode", exitCode)
	r.sink.OnFinish(runID, status, exitCode, errMsg)
}

// pump membaca satu stream baris demi baris dan meneruskannya ke sink.
func (r *Runner) pump(runID string, rc interface{ Read([]byte) (int, error) }, stream model.LogStream) {
	scanner := bufio.NewScanner(rc)
	// Baris log CrewAI bisa panjang (satu prompt utuh). Default 64KB kurang,
	// dan kalau terlampaui scanner berhenti diam-diam di tengah run.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if stream == model.StreamStdout {
			if m := branchPattern.FindStringSubmatch(line); m != nil {
				r.sink.OnBranch(runID, m[1])
			}
		}
		r.sink.OnLine(runID, stream, line)
	}
	if err := scanner.Err(); err != nil {
		r.sink.OnLine(runID, model.StreamSystem, fmt.Sprintf("[api] gagal membaca %s: %v", stream, err))
	}
}

// classify menerjemahkan hasil cmd.Wait jadi status domain.
func classify(ctx context.Context, waitErr error) (model.RunStatus, int, string) {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return model.RunError, -1, "Run dihentikan karena melewati batas waktu."
	case errors.Is(ctx.Err(), context.Canceled):
		return model.RunCancelled, -1, "Run dibatalkan."
	case waitErr == nil:
		// Engine exit 0 berarti SQA lulus, atau lulus tapi deploy dilewati.
		return model.RunPassed, 0, ""
	}

	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		// Engine keluar dengan exit code. Yang paling sering exit 1, dan itu
		// punya dua arti yang tidak bisa dibedakan dari luar: SQA masih FAIL
		// setelah semua ronde (hasil yang sah), atau Python mati karena
		// exception yang tidak tertangani, misalnya endpoint LLM tidak bisa
		// dihubungi. Keduanya dipetakan ke FAILED; log run yang membedakannya.
		return model.RunFailed, exitErr.ExitCode(), ""
	}
	return model.RunError, -1, fmt.Sprintf("engine berhenti tidak wajar: %v", waitErr)
}

// Shutdown membatalkan semua run yang masih jalan. Dipanggil saat API berhenti.
func (r *Runner) Shutdown(timeout time.Duration) {
	r.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(r.running))
	for _, c := range r.running {
		cancels = append(cancels, c)
	}
	r.mu.Unlock()

	for _, c := range cancels {
		c()
	}

	deadline := time.After(timeout)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			slog.Warn("masih ada run yang belum berhenti saat shutdown")
			return
		case <-tick.C:
			r.mu.Lock()
			n := len(r.running)
			r.mu.Unlock()
			if n == 0 {
				return
			}
		}
	}
}
