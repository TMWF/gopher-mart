package service

import "log/slog"

type OrdersService interface {
}

type ordersService struct {
	logger *slog.Logger
	//TODO: Add Repository
}

func NewOrdersService(logger *slog.Logger) *ordersService {
	return &ordersService{
		logger: logger.With(slog.String("op", "service.OrdersService")),
	}
}
