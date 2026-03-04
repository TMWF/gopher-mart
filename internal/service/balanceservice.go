package service

import (
	"log/slog"

	"github.com/TMWF/gopher-mart/internal/repository"
)

type BalanceService interface{}

type balanceService struct {
	logger     *slog.Logger
	repository repository.BalanceRepository
}

func NewBalanceService(logger *slog.Logger) *balanceService {
	return &balanceService{logger: logger}
}
