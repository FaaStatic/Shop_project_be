package cmd

import (
	"fmt"
	"os"
	configdb "shop_project_be/internal/config/config_db"
	envconfig "shop_project_be/internal/config/env_config"
	"shop_project_be/internal/domain"
	loggerconfig "shop_project_be/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var migrateResetCmd = &cobra.Command{
	Use:   "migrate-reset",
	Short: "Delete All table and do re-migrate",
	Run: func(cmd *cobra.Command, args []string) {
		env := os.Getenv("APP_ENV")
		log := loggerconfig.LoggerCustom(env)
		defer log.Sync()
		envConf, err := envconfig.InitEnvConfig(log)
		if err != nil {
			log.Fatal("Failed to initialize environment config: %v", zap.Error(err))
		}
		db, err := configdb.InitDB(envConf.DB, log, env)
		if err != nil {
			log.Fatal("Failed to initialize database:", zap.Error(err))
		}
		err = db.Migrator().DropTable(&domain.Users{}, &domain.Customers{}, &domain.Products{}, &domain.Debts{}, &domain.DebtPayments{}, &domain.Transactions{}, &domain.TransactionsDetail{})
		if err != nil {
			log.Fatal("Gagal mereset database: %v", zap.Error(err))
			panic("stop! Failed to reset database")
		}

		sqldb, err := db.DB()
		if err != nil {
			log.Fatal("Failed to get sql.DB:", zap.Error(err))
			panic(fmt.Sprintf("stop! Failed to reset database"))
		}
		err = sqldb.Close()
		if err != nil {
			log.Fatal("Failed to close database connection:", zap.Error(err))
			panic(fmt.Sprintf("stop! Failed to reset database"))
		}

		configdb.MigrateDB(log)
		fmt.Println("Database berhasil di-reset dan di-migrasi ulang!")
	},
}

func init() {
	rootCmd.AddCommand(migrateResetCmd)
}
