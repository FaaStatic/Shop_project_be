package cmd

import (
	"context"
	"os"
	envconfig "shop_project_be/config/env_config"
	"shop_project_be/infrastructure/database"
	loggerconfig "shop_project_be/infrastructure/logger"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/domain"
	"shop_project_be/internal/repository"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	createAdminUsername string
	createAdminPassword string
	createAdminRole     string
)

// createAdmin membuat akun admin/superadmin langsung ke database. Endpoint
// register publik hanya untuk staff, jadi akun privileged dibuat lewat CLI ini.
var createAdmin = &cobra.Command{
	Use:   "create-admin",
	Short: "Membuat akun admin atau superadmin langsung ke database",
	Run: func(cmd *cobra.Command, args []string) {
		env := os.Getenv("APP_ENV")
		loggerconfig.LoggerCustom(env)
		defer loggerconfig.Logger.Sync()

		// CLI hanya untuk akun superadmin; staff lewat endpoint register.
		if createAdminRole != "superadmin" {
			loggerconfig.Logger.Fatal("role harus 'superadmin'", zap.String("role", createAdminRole))
		}
		roleEnum, err := enum.ParseUserRole(createAdminRole)
		if err != nil {
			loggerconfig.Logger.Fatal("role tidak valid", zap.Error(err))
		}
		if len(createAdminPassword) < 6 {
			loggerconfig.Logger.Fatal("password minimal 6 karakter")
		}

		envConf, err := envconfig.InitEnvConfig(loggerconfig.Logger)
		if err != nil {
			loggerconfig.Logger.Fatal("gagal init config", zap.Error(err))
		}

		db, err := database.InitDB(envConf.DB, loggerconfig.Logger, env)
		if err != nil {
			loggerconfig.Logger.Fatal("gagal init database", zap.Error(err))
		}

		userRepo := repository.NewUserRepository(db)
		ctx := context.Background()

		existing, err := userRepo.GetUserByUsername(ctx, createAdminUsername)
		if err != nil {
			loggerconfig.Logger.Fatal("gagal cek username", zap.Error(err))
		}
		if existing != nil {
			loggerconfig.Logger.Fatal("username sudah dipakai", zap.String("username", createAdminUsername))
		}

		user := &domain.Users{
			Username: createAdminUsername,
			Password: createAdminPassword,
			Role:     roleEnum,
		}
		if err := user.HashPswd(); err != nil {
			loggerconfig.Logger.Fatal("gagal hash password", zap.Error(err))
		}
		if err := userRepo.RegisterUser(ctx, user); err != nil {
			loggerconfig.Logger.Fatal("gagal membuat user", zap.Error(err))
		}

		loggerconfig.Logger.Info("akun berhasil dibuat",
			zap.String("username", createAdminUsername),
			zap.String("role", roleEnum.String()),
		)
	},
}

func init() {
	createAdmin.Flags().StringVarP(&createAdminUsername, "username", "u", "", "username akun (wajib)")
	createAdmin.Flags().StringVarP(&createAdminPassword, "password", "p", "", "password akun, minimal 6 karakter (wajib)")
	createAdmin.Flags().StringVarP(&createAdminRole, "role", "r", "superadmin", "role akun (hanya superadmin)")
	createAdmin.MarkFlagRequired("username")
	createAdmin.MarkFlagRequired("password")
	rootCmd.AddCommand(createAdmin)
}
