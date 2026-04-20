package cmd

import (
	"os"
	envconfig "shop_project_be/internal/config/env_config"
	fiberconfig "shop_project_be/internal/config/fiber_config"
	"shop_project_be/pkg/cache"
	loggerconfig "shop_project_be/pkg/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var serverRun = &cobra.Command{
	Use:   "serve",
	Short: "Running Server Based Fiber",
	Run: func(cmd *cobra.Command, args []string) {
		env := os.Getenv("APP_ENV")
		zapLogger := loggerconfig.LoggerCustom(env)
		defer zapLogger.Sync()

		envConf, err := envconfig.InitEnvConfig(zapLogger)
		if err != nil {
			zapLogger.Fatal("gagal init config", zap.Error(err))
		}

		redisClient, err := cache.InitRedis(&envConf.Redis)
		if err != nil {
			zapLogger.Fatal("gagal init redis", zap.Error(err))
		}
		defer redisClient.Close()

		app := fiberconfig.InitFiber(env, envConf, zapLogger)

		zapLogger.Info("server starting",
			zap.String("app", envConf.App.Name),
			zap.String("port", envConf.App.Port),
		)

		if err := app.Listen(":"+envConf.App.Port, fiberconfig.GetFiberConfigListener(envConf.App.Env)); err != nil {
			zapLogger.Fatal("server error", zap.Error(err))
		}
	},
}

func init() {
	rootCmd.AddCommand(serverRun)
}
