package database

import (
	envconfig "shop_project_be/config/env_config"
	zaplogger "shop_project_be/infrastructure/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
)

func InitDB(config envconfig.DBConfig, log *zap.Logger, env string) (*gorm.DB, error) {
	dsn := "host=" + config.Host + " user=" + config.User + " password=" + config.Password + " dbname=" + config.DBName + " port=" + config.Port + " sslmode=" + config.SSLMode + " TimeZone=" + config.TimeZone

	gormLog := zaplogger.NewGormZapLogger(log)
	usingPooler := false
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: usingPooler,
	}), &gorm.Config{
		Logger:                 gormLog,
		PrepareStmt:            !usingPooler,
		SkipDefaultTransaction: true,
		TranslateError:         true,
	})

	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("Failed to get sql.DB!")
	}

	// Pool sizing: use config overrides when provided, else the previous
	// defaults (unchanged behavior). Under prefork each process owns a pool, so
	// max_open_conns should be tuned so N processes stay within the database's
	// max_connections.
	maxOpen := 100
	if config.MaxOpenConns > 0 {
		maxOpen = config.MaxOpenConns
	}
	maxIdle := 10
	if config.MaxIdleConns > 0 {
		maxIdle = config.MaxIdleConns
	}
	connLifetime := time.Hour
	if config.ConnMaxLifetimeMinutes > 0 {
		connLifetime = time.Duration(config.ConnMaxLifetimeMinutes) * time.Minute
	}
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(connLifetime)
	log.Info("Database PostgreSQL Connected",
		zap.String("host", config.Host),
		zap.Int("max_open_conns", maxOpen),
		zap.Int("max_idle_conns", maxIdle),
	)

	return db, nil

}
