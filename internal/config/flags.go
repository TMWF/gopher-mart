package config

import (
	"flag"
	"log/slog"
	"time"

	"github.com/TMWF/gopher-mart/internal/config/db"
	myLoggerPackage "github.com/TMWF/gopher-mart/internal/logger"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	SecretKey            string `env:"SECRET_KEY"`
	TokenExp             time.Duration
	// BaseURL        string `env:"BASE_URL"`
	// LogLevel       string `env:"LOG_LEVEL"`
	db.PostgreSQLConfig
}

func InitialiseConfigs(logger *slog.Logger) *Config {
	log := logger.With(slog.String("op", "config.InitialiseConfigs"))

	cfg := &Config{}
	var serverHost string
	var accrualSystemAddress string
	// var baseURLFlag string
	// var logLevel string
	// var urlStoragePath string
	var databaseDSN string
	var tokenExp int
	var secretKey string

	flag.StringVar(&serverHost, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&accrualSystemAddress, "r", "", "accrual system address")
	// flag.StringVar(&baseURLFlag, "b", "http://localhost:8080", "address and port to run server")
	// flag.StringVar(&logLevel, "c", "DEBUG", "logging level")
	// flag.StringVar(&urlStoragePath, "f", "", "File storage path for urls")
	flag.StringVar(&databaseDSN, "d", "postgres://admin:admin@localhost:5432/postgres?sslmode=disable", "PostgreSQL DSN")
	flag.IntVar(&tokenExp, "t", 3, "Token expiration in hours")
	flag.StringVar(&secretKey, "s", "supersecretkey", "JWT secret key")
	flag.Parse()

	err := env.Parse(cfg)
	if err != nil {
		log.Error("Error occured while parsing environment", myLoggerPackage.Err(err))
	}

	if cfg.RunAddress == "" {
		cfg.RunAddress = serverHost
	}
	log.Debug("Run address: " + cfg.RunAddress)

	if cfg.AccrualSystemAddress == "" {
		cfg.AccrualSystemAddress = accrualSystemAddress
	}
	log.Debug("Accrual: " + cfg.AccrualSystemAddress)

	// if cfg.BaseURL == "" {
	// 	cfg.BaseURL = baseURLFlag
	// }

	// if cfg.LogLevel == "" {
	// 	cfg.LogLevel = logLevel
	// }

	if cfg.DatabaseDSN == "" {
		log.Debug("Setting database config")
		cfg.DatabaseDSN = databaseDSN
	}
	log.Debug("DatabaseDSN: " + cfg.DatabaseDSN)

	if cfg.SecretKey == "" {
		cfg.SecretKey = secretKey
	}

	cfg.TokenExp = time.Duration(tokenExp) * time.Hour

	return cfg
}
