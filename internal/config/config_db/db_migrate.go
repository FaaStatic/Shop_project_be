package configdb

import (
	"fmt"
	envconfig "shop_project_be/internal/config/env_config"
	"shop_project_be/internal/domain"

	"go.uber.org/zap"
)

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

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Warn("gagal buat extension uuid-ossp", zap.Error(err))
	}

	entities := []interface{}{
		&domain.Users{},
		&domain.Customers{},
		&domain.Products{},
		&domain.Debts{},
		&domain.DebtPayments{},
		&domain.Transactions{},
		&domain.TransactionsDetail{},
	}

	if err := db.AutoMigrate(entities...); err != nil {
		return fmt.Errorf("gagal migrate: %w", err)
	}

	log.Info("migrasi berhasil",
		zap.Int("total_tabel", len(entities)),
	)
	return nil
}
