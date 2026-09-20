package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"

	"github.com/alexistdev/geosquad/api/migrations"
)

// File migrasi ikut ditanam ke dalam binary lewat package migrations.
// Konsekuensinya satu binary cukup untuk deploy: tidak ada folder migrations
// yang harus ikut dikirim, dan tidak ada versi skema yang tertinggal di server.

func prepare() error {
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("set dialect goose: %w", err)
	}
	return nil
}

// Migrate menjalankan semua migrasi yang belum pernah jalan. Idempotent:
// goose mencatat versi yang sudah diterapkan di tabel goose_db_version.
func Migrate(conn *sql.DB) error {
	if err := prepare(); err != nil {
		return err
	}

	before, _ := goose.GetDBVersion(conn)
	if err := goose.Up(conn, migrations.Dir); err != nil {
		return fmt.Errorf("jalankan migrasi: %w", err)
	}
	after, err := goose.GetDBVersion(conn)
	if err != nil {
		return fmt.Errorf("baca versi skema: %w", err)
	}

	if before == after {
		slog.Info("skema database sudah terkini", "versi", after)
	} else {
		slog.Info("migrasi diterapkan", "dari", before, "ke", after)
	}
	return nil
}

// MigrateDown memundurkan satu versi. Dipakai lewat flag, tidak pernah otomatis.
func MigrateDown(conn *sql.DB) error {
	if err := prepare(); err != nil {
		return err
	}
	return goose.Down(conn, migrations.Dir)
}

// MigrateStatus mencetak versi skema saat ini.
func MigrateStatus(conn *sql.DB) error {
	if err := prepare(); err != nil {
		return err
	}
	return goose.Status(conn, migrations.Dir)
}
