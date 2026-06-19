package database

import (
	"fmt"
	envconfig "shop_project_be/config/env_config"

	"go.uber.org/zap"
)

// MigrateDB menerapkan migration versioned (goose) ke database. Skema tidak lagi
// dikelola lewat GORM AutoMigrate; setiap perubahan struktur harus ditambahkan
// sebagai file SQL baru di infrastructure/database/migrations.
func MigrateDB(log *zap.Logger) error {
	envConf, err := envconfig.InitEnvConfig(log)
	if err != nil {
		return fmt.Errorf("gagal init config: %w", err)
	}

	db, err := InitDB(envConf.DB, log, envConf.App.Env)
	if err != nil {
		return fmt.Errorf("gagal init database: %w", err)
	}

	sqlDb, err := db.DB()
	if err != nil {
		return fmt.Errorf("gagal ambil sql.DB: %w", err)
	}
	defer sqlDb.Close()

	if err := RunMigrations(sqlDb, log); err != nil {
		return err
	}

	log.Info("migrasi berhasil")
	return nil
}
