package cmd

import (
	"os"
	envconfig "shop_project_be/config/env_config"
	fiberconfig "shop_project_be/config/fiber_config"
	"shop_project_be/infrastructure/cache"
	loggerconfig "shop_project_be/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var serverRun = &cobra.Command{
	Use:   "serve",
	Short: "Running Server Based Fiber",
	Run: func(cmd *cobra.Command, args []string) {
		env := os.Getenv("APP_ENV")
		loggerconfig.LoggerCustom(env)
		defer loggerconfig.Logger.Sync()

		envConf, err := envconfig.InitEnvConfig(loggerconfig.Logger)
		if err != nil {
			loggerconfig.Logger.Fatal("gagal init config", zap.Error(err))
		}

		redisClient, err := cache.InitRedis(&envConf.Redis)
		if err != nil {
			loggerconfig.Logger.Fatal("gagal init redis", zap.Error(err))
		}
		defer redisClient.Close()

		app := fiberconfig.InitFiber(env, envConf, loggerconfig.Logger)

		loggerconfig.Logger.Info("server starting",
			zap.String("app", envConf.App.Name),
			zap.String("port", envConf.App.Port),
		)

		if err := app.Listen(":"+envConf.App.Port, fiberconfig.GetFiberConfigListener(envConf.App.Env)); err != nil {
			loggerconfig.Logger.Fatal("server error", zap.Error(err))
		}
	},
}

func init() {
	rootCmd.AddCommand(serverRun)
}
