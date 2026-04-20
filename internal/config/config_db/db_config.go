package configdb

import (
	envconfig "shop_project_be/internal/config/env_config"
	"time"

	loggerconfig "shop_project_be/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	gormlogger "gorm.io/gorm/logger"

	"gorm.io/gorm"
)

func InitDB(config envconfig.DBConfig, log *zap.Logger, env string) (*gorm.DB, error) {
	dsn := "host=" + config.Host + " user=" + config.User + " password=" + config.Password + " dbname=" + config.DBName + " port=" + config.Port + " sslmode=" + config.SSLMode + " TimeZone=" + config.TimeZone

	gormLog := loggerconfig.NewGormZapLogger(log)

	if env == "production" {
		gormLog = gormLog.LogMode(gormlogger.Error).(*loggerconfig.GormZapLogger)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormLog.LogLevel),
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})

	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("Failed to get sql.DB!")
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil

}
