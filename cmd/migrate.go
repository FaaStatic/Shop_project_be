package cmd

import (
	"os"
	configdb "shop_project_be/internal/config/config_db"
	loggerconfig "shop_project_be/pkg/logger"

	"github.com/spf13/cobra"
)

var migrateDb = &cobra.Command{
	Use:   "migrate",
	Short: "Shop Migrate Database",
	Run: func(cmd *cobra.Command, args []string) {
		env := os.Getenv("APP_ENV")

		log := loggerconfig.LoggerCustom(env)
		defer log.Sync()
		configdb.MigrateDB(log)
	},
}

func init() {
	rootCmd.AddCommand(migrateDb)
}
