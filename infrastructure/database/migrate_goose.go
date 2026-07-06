package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

// migrationFS embeds all SQL migration files into the binary so
// migrations do not depend on file paths at runtime/deploy.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

const migrationsDir = "migrations"

// gooseZapLogger bridges goose logging to zap for consistency with the application
// application (see also GormZapLogger in infrastructure/logger).
type gooseZapLogger struct {
	log *zap.Logger
}

func (l *gooseZapLogger) Printf(format string, v ...interface{}) {
	l.log.Sugar().Infof(format, v...)
}

func (l *gooseZapLogger) Fatalf(format string, v ...interface{}) {
	l.log.Sugar().Fatalf(format, v...)
}

// setupGoose prepares goose: file source from the embed FS, postgres dialect, and
// zap logger. Called once before any migration operation.
func setupGoose(log *zap.Logger) error {
	goose.SetBaseFS(migrationFS)
	goose.SetLogger(&gooseZapLogger{log: log})
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}
	return nil
}

// RunMigrations applies all pending migrations (goose up).
func RunMigrations(db *sql.DB, log *zap.Logger) error {
	if err := setupGoose(log); err != nil {
		return err
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}
	return nil
}

// ResetMigrations rolls back all migrations (drops all tables) then
// runs them again from scratch. Used by the migrate-reset command.
func ResetMigrations(db *sql.DB, log *zap.Logger) error {
	if err := setupGoose(log); err != nil {
		return err
	}
	if err := goose.DownTo(db, migrationsDir, 0); err != nil {
		return fmt.Errorf("failed to reset migration: %w", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to re-migrate: %w", err)
	}
	return nil
}
