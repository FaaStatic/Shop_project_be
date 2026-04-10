package cmd

import (
	configdb "shop_project_be/internal/config/config_db"

	"github.com/spf13/cobra"
)

var migrateDb = &cobra.Command{
	Use:   "migrate",
	Short: "Shop Migrate Database",
	Run: func(cmd *cobra.Command, args []string) {
		configdb.MigrateDB()
	},
}

func init() {
	rootCmd.AddCommand(migrateDb)
}
