package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

// migrationFS menyertakan seluruh file SQL migration ke dalam binary sehingga
// migrasi tidak bergantung pada path file saat runtime/deploy.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

const migrationsDir = "migrations"

// gooseZapLogger menjembatani logging goose ke zap agar seragam dengan logger
// aplikasi (lihat juga GormZapLogger di infrastructure/logger).
type gooseZapLogger struct {
	log *zap.Logger
}

func (l *gooseZapLogger) Printf(format string, v ...interface{}) {
	l.log.Sugar().Infof(format, v...)
}

func (l *gooseZapLogger) Fatalf(format string, v ...interface{}) {
	l.log.Sugar().Fatalf(format, v...)
}

// setupGoose menyiapkan goose: sumber file dari embed FS, dialect postgres, dan
// logger zap. Dipanggil sekali sebelum operasi migrasi.
func setupGoose(log *zap.Logger) error {
	goose.SetBaseFS(migrationFS)
	goose.SetLogger(&gooseZapLogger{log: log})
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("gagal set dialect goose: %w", err)
	}
	return nil
}

// RunMigrations menerapkan seluruh migration yang belum dijalankan (goose up).
func RunMigrations(db *sql.DB, log *zap.Logger) error {
	if err := setupGoose(log); err != nil {
		return err
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("gagal menjalankan migration: %w", err)
	}
	return nil
}

// ResetMigrations menurunkan seluruh migration (drop semua tabel) lalu
// menjalankannya kembali dari awal. Dipakai oleh command migrate-reset.
func ResetMigrations(db *sql.DB, log *zap.Logger) error {
	if err := setupGoose(log); err != nil {
		return err
	}
	if err := goose.DownTo(db, migrationsDir, 0); err != nil {
		return fmt.Errorf("gagal reset migration: %w", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("gagal re-migrate: %w", err)
	}
	return nil
}
