package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/TMWF/gopher-mart/internal/config"
	"github.com/TMWF/gopher-mart/internal/database"
	"github.com/TMWF/gopher-mart/internal/handler"
	mylogger "github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/middleware"
	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/TMWF/gopher-mart/internal/service"
	"github.com/TMWF/gopher-mart/internal/util/validation"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	run()
}

func run() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}
	logger := mylogger.SetupLogger(env)
	cfg := config.InitialiseConfigs(logger)
	pgxpool, err := database.NewPostgresPool(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		logger.Error("failed to initialise pgxpool server", mylogger.Err(err))
		return
	}

	defer pgxpool.Close()

	logger.Info("starting application", slog.String("env", env))
	router := createRouter(cfg, logger, pgxpool)
	logger.Info("startingServer", slog.String("address", cfg.RunAddress))
	if err := http.ListenAndServe(cfg.RunAddress, router); err != nil {
		log.Fatal("failed to start server", err)
	}
}

// TODO: добавить к параметрам метода БД
func createRouter(cfg *config.Config, logger *slog.Logger, pgxpool *pgxpool.Pool) http.Handler {
	v := validator.New()
	err := v.RegisterValidation("luhn", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return validation.IsLuhnValid(value)
	})

	if err != nil {
		log.Fatal("Error ocured while tryiong to register validator")
	}

	userService := service.NewUserService(logger)
	userHandler := handler.NewUserHandler(userService, logger)

	balanceRepository := repository.NewBalanceRepository(logger, pgxpool)
	balanceService := service.NewBalanceService(logger, balanceRepository)
	balanceHandler := handler.NewBalanceHandler(logger, balanceService, v)

	ordersRepository := repository.NewOrdersRepository(pgxpool, logger)
	ordersService := service.NewOrdersService(logger, ordersRepository)
	ordersHandler := handler.NewOrdersHandler(logger, ordersService)

	router := chi.NewRouter()

	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.Recoverer)
	router.Use(middleware.NewRequestLogger(logger))
	router.Use(middleware.GzipMiddleware())
	router.Use(middleware.JwtTokenMiddleware(cfg, logger))
	router.Post(`/api/user/register`, userHandler.RegisterUser)
	router.Post(`/api/user/login`, userHandler.LoginUser)
	router.Get(`/api/user/balance`, balanceHandler.GetBalanceForUser)
	router.Post(`/api/user/balance/withdraw`, balanceHandler.WithdrawForOrder)
	router.Post(`/api/user/orders`, ordersHandler.UploadOrder)
	router.Get(`/api/user/orders`, ordersHandler.GetUserOrders)
	return router
}
