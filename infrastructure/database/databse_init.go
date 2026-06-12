package database

import (
	envconfig "shop_project_be/config/env_config"
	zaplogger "shop_project_be/infrastructure/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	gormlogger "gorm.io/gorm/logger"

	"gorm.io/gorm"
)

func InitDB(config envconfig.DBConfig, log *zap.Logger, env string) (*gorm.DB, error) {
	dsn := "host=" + config.Host + " user=" + config.User + " password=" + config.Password + " dbname=" + config.DBName + " port=" + config.Port + " sslmode=" + config.SSLMode + " TimeZone=" + config.TimeZone

	gormLog := zaplogger.NewGormZapLogger(log)

	if env == "production" {
		gormLog = gormLog.LogMode(gormlogger.Error).(*zaplogger.GormZapLogger)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormLog.LogLevel),
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
		// TranslateError menerjemahkan error driver menjadi error GORM yang
		// generik (mis. unique violation -> gorm.ErrDuplicatedKey), sehingga
		// repository bisa mendeteksinya tanpa bergantung pada kode error Postgres.
		TranslateError: true,
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
	log.Info("Database PostgreSQL Connected", zap.String("host", config.Host))

	return db, nil

}
