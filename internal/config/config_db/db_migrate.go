package configdb

import "shop_project_be/internal/domain"

func MigrateDB() {
	db := InitDB()
	sqlDb, err := db.DB()
	if err != nil {
		panic("Failed to open db!")
	}
	errMigrate := db.AutoMigrate(&domain.Users{}, &domain.Products{}, &domain.Transactions{}, &domain.Customers{}, &domain.Debts{}, &domain.DebtPayments{}, &domain.TransactionsDetail{})
	if errMigrate != nil {
		panic("Failed to migrate database!")
	}
	defer sqlDb.Close()

}
