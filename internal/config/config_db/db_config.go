package configdb

import (
	"shop_project_be/internal/domain"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	envDb, err := godotenv.Read()
	if err != nil {
		panic("Failed Load .env file!")
	}

	dsn := "host=" + envDb["DB_HOST"] + " user=" + envDb["DB_USER"] + " password=" + envDb["DB_PASSWORD"] + " dbname=" + envDb["DB_NAME"] + " port=" + envDb["DB_PORT"] + " sslmode=disable TimeZone=Asia/Jakarta"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed Connect to Database!")
	}
	err = db.AutoMigrate(&domain.Users{}, &domain.Products{}, &domain.Transactions{}, &domain.Customers{}, &domain.Debts{}, &domain.DebtPayments{}, &domain.TransactionsDetail{})
	if err != nil {
		panic("Failed to migrate database!")
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("Failed to get sql.DB!")
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db

}
