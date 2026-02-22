package config

import (
	"flag"
	"log/slog"

	myLoggerPackage "github.com/TMWF/gopher-mart/internal/logger"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddress string `env:"RUN_ADDRESS"`
	// BaseURL        string `env:"BASE_URL"`
	// LogLevel       string `env:"LOG_LEVEL"`
	// db.PostgreSQLConfig
	// UserJWTConfig
}

func InitialiseConfigs(logger *slog.Logger) *Config {
	log := logger.With(slog.String("op", "config.InitialiseConfigs"))

	cfg := &Config{}
	var serverHostFlag string
	// var baseURLFlag string
	// var logLevel string
	// var urlStoragePath string
	// var databaseDSN string
	// var tokenExp int
	// var secretKey string

	flag.StringVar(&serverHostFlag, "a", "localhost:8080", "address and port to run server")
	// flag.StringVar(&baseURLFlag, "b", "http://localhost:8080", "address and port to run server")
	// flag.StringVar(&logLevel, "c", "DEBUG", "logging level")
	// flag.StringVar(&urlStoragePath, "f", "", "File storage path for urls")
	// flag.StringVar(&databaseDSN, "d", "", "PostgreSQL DSN")
	// flag.IntVar(&tokenExp, "t", 3, "Token expiration in hours")
	// flag.StringVar(&secretKey, "s", "supersecretkey", "JWT secret key")
	// flag.Parse()

	err := env.Parse(cfg)
	if err != nil {
		log.Error("Error occured while parsing environment", myLoggerPackage.Err(err))
	}

	if cfg.RunAddress == "" {
		cfg.RunAddress = serverHostFlag
	}

	// if cfg.BaseURL == "" {
	// 	cfg.BaseURL = baseURLFlag
	// }

	// if cfg.LogLevel == "" {
	// 	cfg.LogLevel = logLevel
	// }

	// if cfg.URLStoragePath == "" {
	// 	cfg.URLStoragePath = urlStoragePath
	// }

	// if cfg.DatabaseDSN == "" {
	// 	logger.GetLogger().Debug("Setting database config")
	// 	cfg.DatabaseDSN = databaseDSN
	// }

	// if cfg.SecretKey == "" {
	// 	cfg.SecretKey = secretKey
	// }

	// cfg.TokenExp = 3 * time.Hour

	return cfg
}
