package envconfig

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	App     AppConfig
	DB      DBConfig
	Redis   RedisConfig
	JWT     JWTConfig
	Encrypt EncryptConfig
}

type AppConfig struct {
	Name string
	Port string
	Env  string
	Host string
	// TrustedProxies adalah daftar IP/CIDR reverse proxy yang dipercaya. Bila
	// diisi (mis. ["127.0.0.1"] saat di belakang nginx), c.IP() akan membaca
	// X-Forwarded-For sehingga rate limiter mengenali IP user asli, bukan IP
	// proxy. Kosong = tidak ada proxy dipercaya (default, anti header spoofing).
	TrustedProxies []string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	TimeZone string
}

type RedisConfig struct {
	URL      string
	Db       int
	Password string
	Port     string
	Host     string
	Username string
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  int
	RefreshTokenTTL int
}

type EncryptConfig struct {
	Key string
}

type configError struct {
	field   string
	message string
}

func (e *configError) Error() string {
	return fmt.Sprintf("config error [%s]: %s", e.field, e.message)
}

func InitEnvConfig(log *zap.Logger) (cfg *Config, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic saat load config: %v", r)
			log.Error("config loader panic", zap.Any("recover", r))
		}
	}()
	env := os.Getenv("APP_ENV")
	if env == "development" {
		viper.SetConfigName(".config.development")
	} else {
		viper.SetConfigName(".config.production")
	}

	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("gagal membaca config file: %w", err)
	}

	cfg = &Config{
		App: AppConfig{
			Name:           viper.GetString("server.name"),
			Port:           viper.GetString("server.port"),
			Env:            viper.GetString("server.env"),
			Host:           viper.GetString("server.host"),
			TrustedProxies: viper.GetStringSlice("server.trusted_proxies"),
		},
		DB: DBConfig{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetString("database.port"),
			User:     viper.GetString("database.user"),
			Password: viper.GetString("database.pass"),
			DBName:   viper.GetString("database.dbname"),
			SSLMode:  viper.GetString("database.sslmode"),
			TimeZone: viper.GetString("database.time_zone"),
		},
		Redis: RedisConfig{
			URL:      viper.GetString("redis.url"),
			Db:       viper.GetInt("redis.db"),
			Password: viper.GetString("redis.password"),
			Port:     viper.GetString("redis.port"),
			Host:     viper.GetString("redis.host"),
			Username: viper.GetString("redis.username"),
		},
		JWT: JWTConfig{
			Secret:          viper.GetString("jwt.secret"),
			AccessTokenTTL:  viper.GetInt("jwt.token_ttl"),
			RefreshTokenTTL: viper.GetInt("jwt.refresh_token_ttl"),
		},
		Encrypt: EncryptConfig{
			Key: viper.GetString("encrypt.key"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	log.Info("config berhasil dimuat",
		zap.String("env", cfg.App.Env),
		zap.String("config_file", viper.ConfigFileUsed()),
	)

	return cfg, nil
}

func (c *Config) validate() error {
	type check struct {
		value string
		field string
	}

	required := []check{
		{c.App.Port, "server.port"},
		{c.App.Env, "server.env"},
		{c.DB.Host, "database.host"},
		{c.DB.Port, "database.port"},
		{c.DB.User, "database.user"},
		{c.DB.Password, "database.password"},
		{c.DB.DBName, "database.dbname"},
		{c.JWT.Secret, "jwt.secret"},
		{c.Encrypt.Key, "encrypt.key"},
	}

	for _, r := range required {
		if r.value == "" {
			return &configError{
				field:   r.field,
				message: "field wajib tidak boleh kosong",
			}
		}
	}
	if c.JWT.AccessTokenTTL <= 0 {
		return &configError{field: "jwt.token_ttl", message: "harus lebih besar dari 0"}
	}
	if c.JWT.RefreshTokenTTL <= 0 {
		return &configError{field: "jwt.refresh_token_ttl", message: "harus lebih besar dari 0"}
	}

	return nil
}
