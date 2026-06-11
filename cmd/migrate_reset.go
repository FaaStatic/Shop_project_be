package cmd

import (
	"fmt"
	"os"
	envconfig "shop_project_be/config/env_config"
	"shop_project_be/infrastructure/database"
	zaplogger "shop_project_be/infrastructure/logger"
	"shop_project_be/internal/domain"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var migrateResetCmd = &cobra.Command{
	Use:   "migrate-reset",
	Short: "Delete All table and do re-migrate",
	Run: func(cmd *cobra.Command, args []string) {
		env := os.Getenv("APP_ENV")
		zaplogger.LoggerCustom(env)
		defer zaplogger.Logger.Sync()
		envConf, err := envconfig.InitEnvConfig(zaplogger.Logger)
		if err != nil {
			zaplogger.Logger.Fatal("Failed to initialize environment config: %v", zap.Error(err))
		}
		db, err := database.InitDB(envConf.DB, zaplogger.Logger, env)
		if err != nil {
			zaplogger.Logger.Fatal("Failed to initialize database:", zap.Error(err))
		}
		err = db.Migrator().DropTable(&domain.Users{}, &domain.Customers{}, &domain.Products{}, &domain.Debts{}, &domain.DebtPayments{}, &domain.Transactions{}, &domain.TransactionsDetail{})
		if err != nil {
			zaplogger.Logger.Fatal("Gagal mereset database: %v", zap.Error(err))
			panic("stop! Failed to reset database")
		}

		sqldb, err := db.DB()
		if err != nil {
			zaplogger.Logger.Fatal("Failed to get sql.DB:", zap.Error(err))
			panic("stop! Failed to reset database")
		}
		err = sqldb.Close()
		if err != nil {
			zaplogger.Logger.Fatal("Failed to close database connection:", zap.Error(err))
			panic("stop! Failed to reset database")
		}

		if err := database.MigrateDB(zaplogger.Logger); err != nil {
			zaplogger.Logger.Fatal("gagal migrate database", zap.Error(err))
		}
		fmt.Println("Database berhasil di-reset dan di-migrasi ulang!")
	},
}

func init() {
	rootCmd.AddCommand(migrateResetCmd)
}
