// Command api adalah titik masuk GeoSquad API.
//
// Tugasnya merakit dependensi, menjalankan migrasi, lalu melayani HTTP sampai
// diminta berhenti. Semua kebijakan ada di package internal; file ini hanya
// menyambungkannya.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexistdev/geosquad/api/internal/auth"
	"github.com/alexistdev/geosquad/api/internal/cache"
	"github.com/alexistdev/geosquad/api/internal/config"
	"github.com/alexistdev/geosquad/api/internal/db"
	"github.com/alexistdev/geosquad/api/internal/handler"
	"github.com/alexistdev/geosquad/api/internal/middleware"
	"github.com/alexistdev/geosquad/api/internal/repository"
	"github.com/alexistdev/geosquad/api/internal/runner"
	"github.com/alexistdev/geosquad/api/internal/service"
)

// version diisi saat build lewat -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var (
		migrateOnly = flag.Bool("migrate", false, "jalankan migrasi lalu keluar")
		rollback    = flag.Bool("migrate-down", false, "mundurkan satu versi migrasi lalu keluar")
		showStatus  = flag.Bool("migrate-status", false, "tampilkan status migrasi lalu keluar")
	)
	flag.Parse()

	if err := run(*migrateOnly, *rollback, *showStatus); err != nil {
		slog.Error("api berhenti", "error", err)
		os.Exit(1)
	}
}

func run(migrateOnly, rollback, showStatus bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("muat konfigurasi: %w", err)
	}
	setupLogger(cfg)

	conn, err := db.Open(cfg.DB)
	if err != nil {
		return err
	}
	defer conn.Close()

	switch {
	case showStatus:
		return db.MigrateStatus(conn)
	case rollback:
		return db.MigrateDown(conn)
	}

	if err := db.Migrate(conn); err != nil {
		return err
	}
	if migrateOnly {
		slog.Info("migrasi selesai, keluar sesuai flag -migrate")
		return nil
	}

	rdb, err := cache.Open(cfg.Redis)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// ---- rakit dependensi ----------------------------------------------
	userRepo := repository.NewUserRepository(conn)
	runRepo := repository.NewRunRepository(conn)
	logRepo := repository.NewRunLogRepository(conn)

	tokens := auth.NewTokenService(cfg.JWT, cfg.AppName)
	sessions := auth.NewSessionStore(rdb, cfg.JWT.SessionTTL)
	hub := service.NewHub()

	authSvc := service.NewAuthService(userRepo, tokens, sessions)
	userSvc := service.NewUserService(userRepo, sessions)
	runSvc := service.NewRunService(runRepo, logRepo, hub)

	// Runner dan service saling membutuhkan: runner melapor ke service lewat
	// Sink, service menyuruh runner lewat Engine. Disambung di sini supaya
	// tidak ada import melingkar antar package.
	engine := runner.New(cfg.Engine, runSvc)
	runSvc.AttachEngine(engine)

	if err := engine.Validate(); err != nil {
		// Tidak fatal: API tetap berguna untuk auth dan melihat riwayat run,
		// meski engine belum disiapkan. Tapi harus terlihat jelas di log.
		slog.Warn("engine belum siap, pembuatan run akan gagal", "error", err)
	}

	// Run yang tercatat masih jalan pasti sisa proses sebelumnya yang sudah
	// ikut mati. Dibereskan sebelum melayani request.
	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	runSvc.ResetStale(startupCtx)
	cancelStartup()

	handlers := handler.Handlers{
		Health: handler.NewHealthHandler(conn, rdb, version),
		Auth:   handler.NewAuthHandler(authSvc, cfg.JWT),
		User:   handler.NewUserHandler(userSvc),
		Run:    handler.NewRunHandler(runSvc, hub),
	}
	authMW := middleware.NewAuth(tokens, sessions, userRepo, cfg.JWT.CookieName)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.NewRouter(cfg, handlers, authMW),
		// WriteTimeout sengaja 0. Endpoint stream SSE hidup selama run
		// berlangsung, dan batas tulis global akan memutusnya di tengah jalan.
		// Batas per-request ditegakkan lewat context masing-masing.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// ---- jalankan dan tunggu sinyal ------------------------------------
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("api mendengarkan",
			"port", cfg.Port, "env", cfg.Env, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server gagal: %w", err)
	case sig := <-stop:
		slog.Info("sinyal diterima, mulai berhenti dengan rapi", "sinyal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Berhenti menerima request baru dulu, baru hentikan run yang jalan.
	// Terbalik urutannya akan membuat request yang sedang dilayani melihat
	// engine mati di tengah.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown http tidak bersih", "error", err)
	}
	engine.Shutdown(15 * time.Second)

	slog.Info("api berhenti")
	return nil
}

func setupLogger(cfg *config.Config) {
	level := slog.LevelDebug
	if cfg.IsProduction() {
		level = slog.LevelInfo
	}

	var h slog.Handler
	if cfg.IsProduction() {
		// JSON supaya bisa dicerna agregator log.
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(h).With("service", cfg.AppName))
}
