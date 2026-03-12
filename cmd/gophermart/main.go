package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/TMWF/gopher-mart/internal/config"
	"github.com/TMWF/gopher-mart/internal/handler"
	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/middleware"
	"github.com/TMWF/gopher-mart/internal/service"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	run()
}

func run() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}
	log := logger.SetupLogger(env)
	cfg := config.InitialiseConfigs(log)

	log.Info("starting application", slog.String("env", env))
	router := createRouter(cfg, log)
	log.Info("startingServer", slog.String("address", cfg.RunAddress))
	if err := http.ListenAndServe(cfg.RunAddress, router); err != nil {
		log.Error("failed to start server", logger.Err(err))
	}
}

// TODO: добавить к параметрам метода БД
func createRouter(cfg *config.Config, logger *slog.Logger) http.Handler {
	userService := service.NewUserService(logger)
	userHandler := handler.NewUserHandler(userService, logger)

	balanceService := service.NewBalanceService(logger)
	balanceHandler := handler.NewBalanceHandler(logger, balanceService)

	ordersService := service.NewOrdersService(logger)
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
