// Package db memegang koneksi MySQL dan menjalankan migrasi.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/alexistdev/geosquad/api/internal/config"
)

func Open(cfg config.DBConfig) (*sql.DB, error) {
	conn, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("buka koneksi mysql: %w", err)
	}

	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// sql.Open tidak benar-benar menyambung. Tanpa ping di sini, database mati
	// baru ketahuan saat request pertama masuk.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping mysql %s:%s/%s: %w", cfg.Host, cfg.Port, cfg.Name, err)
	}

	slog.Info("mysql tersambung", "host", cfg.Host, "port", cfg.Port, "database", cfg.Name)
	return conn, nil
}
