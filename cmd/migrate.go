package cmd

import (
	"os"
	"shop_project_be/infrastructure/database"
	zaplogger "shop_project_be/infrastructure/logger"

	"github.com/spf13/cobra"
)

var migrateDb = &cobra.Command{
	Use:   "migrate",
	Short: "Shop Migrate Database",
	Run: func(cmd *cobra.Command, args []string) {
		env := os.Getenv("APP_ENV")

		zaplogger.LoggerCustom(env)
		defer zaplogger.Logger.Sync()
		database.MigrateDB(zaplogger.Logger)
	},
}

func init() {
	rootCmd.AddCommand(migrateDb)
}
