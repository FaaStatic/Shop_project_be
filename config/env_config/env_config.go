package envconfig

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	App         AppConfig
	DB          DBConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Encrypt     EncryptConfig
	FirebaseStr FirebaseAppStr
	Midtrans    MidtransConfig
}

type FirebaseAppStr struct {
	GOOGLE_APPLICATION_CREDENTIALS string
}

type AppConfig struct {
	Name string
	Port string
	Env  string
	Host string
	// TrustedProxies is the list of trusted reverse-proxy IPs/CIDRs. If
	// set (e.g. ["127.0.0.1"] when behind nginx), c.IP() reads
	// X-Forwarded-For so the rate limiter sees the real user IP, not the
	// proxy IP. Empty = no trusted proxy (default, anti header spoofing).
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

	// Connection pool sizing. Optional — when unset (0) InitDB keeps the current
	// defaults (MaxOpen 100, MaxIdle 10, lifetime 60m). These matter under
	// prefork: each process owns its own pool, so N processes open up to
	// N×MaxOpen connections. Lower MaxOpenConns to stay within the database's
	// max_connections when running with prefork on multiple cores.
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
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

type MidtransConfig struct {
	ServerKey   string
	ClientKey   string
	Environment string // "sandbox" | "production"
}

// Configured reports whether Midtrans credentials are present. Both keys are
// optional at the config level so the app can boot without online payments;
// callers must gate payment routes on this instead of failing startup.
func (m MidtransConfig) Configured() bool {
	return m.ServerKey != "" && m.ClientKey != ""
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
			err = fmt.Errorf("panic while loading config: %v", r)
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
		return nil, fmt.Errorf("failed to read config file: %w", err)
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
			// Optional pool overrides; 0 (absent) keeps InitDB's defaults.
			MaxOpenConns:           viper.GetInt("database.max_open_conns"),
			MaxIdleConns:           viper.GetInt("database.max_idle_conns"),
			ConnMaxLifetimeMinutes: viper.GetInt("database.conn_max_lifetime_minutes"),
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
		Midtrans: MidtransConfig{
			ServerKey:   viper.GetString("midtrans.server_key"),
			ClientKey:   viper.GetString("midtrans.client_key"),
			Environment: viper.GetString("midtrans.environment"),
		},
		FirebaseStr: FirebaseAppStr{
			GOOGLE_APPLICATION_CREDENTIALS: viper.GetString("firebase.google_application_credentials"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	log.Info("config loaded successfully",
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
		{c.FirebaseStr.GOOGLE_APPLICATION_CREDENTIALS, "firebase.google_application_credentials"},
	}

	for _, r := range required {
		if r.value == "" {
			return &configError{
				field:   r.field,
				message: "required field must not be empty",
			}
		}
	}
	if c.JWT.AccessTokenTTL <= 0 {
		return &configError{field: "jwt.token_ttl", message: "must be greater than 0"}
	}
	if c.JWT.RefreshTokenTTL <= 0 {
		return &configError{field: "jwt.refresh_token_ttl", message: "must be greater than 0"}
	}

	return nil
}
