package cmd

import (
	"os"
	envconfig "shop_project_be/config/env_config"
	fiberconfig "shop_project_be/config/fiber_config"
	"shop_project_be/infrastructure/cache"
	"shop_project_be/infrastructure/database"
	loggerconfig "shop_project_be/infrastructure/logger"
	"shop_project_be/internal/delivery/http/handler"
	"shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/internal/delivery/http/route"
	"shop_project_be/internal/repository"
	"shop_project_be/internal/usecase"
	"shop_project_be/pkg/jwt"

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

		db, err := database.InitDB(envConf.DB, loggerconfig.Logger, env)
		if err != nil {
			loggerconfig.Logger.Fatal("gagal init database", zap.Error(err))
		}

		// Repository
		productRepo := repository.NewProductRepository(db)
		trxRepo := repository.NewTransactionRepository(db)
		userRepo := repository.NewUserRepository(db)
		customerRepo := repository.NewCustomerRepository(db)
		debtRepo := repository.NewDebtRepository(db)
		sessionRepo := repository.NewSessionRepository(redisClient)

		// Service
		jwtService := jwt.NewJWTService(envConf.JWT.Secret, envConf.JWT.AccessTokenTTL, envConf.JWT.RefreshTokenTTL)

		// Usecase
		productUC := usecase.NewProductUsecase(productRepo, loggerconfig.Logger)
		trxUC := usecase.NewTransactionUsecase(trxRepo, productRepo, userRepo, customerRepo, debtRepo, envConf.App.Name, loggerconfig.Logger)
		customerUC := usecase.NewCustomerUsecase(customerRepo, loggerconfig.Logger)
		debtUC := usecase.NewDebtUsecase(debtRepo, loggerconfig.Logger)
		userUC := usecase.NewUserUsecase(userRepo, sessionRepo, loggerconfig.Logger, jwtService)

		// Handler & middleware
		handlers := route.Handlers{
			User:        handler.NewUserHandler(userUC, loggerconfig.Logger),
			Product:     handler.NewProductHandler(productUC, loggerconfig.Logger),
			Transaction: handler.NewTransactionHandler(trxUC, loggerconfig.Logger),
			Customer:    handler.NewCustomerHandler(customerUC, loggerconfig.Logger),
			Debt:        handler.NewDebtHandler(debtUC, loggerconfig.Logger),
		}
		jwtMw := middleware.NewJwtMiddleware(jwtService, sessionRepo)

		app := fiberconfig.InitFiber(env, envConf, loggerconfig.Logger, route.New(handlers, jwtMw, loggerconfig.Logger))

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
