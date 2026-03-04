package repository

import "log/slog"

type BalanceRepository interface {}

type balanceRepository struct {
	logger *slog.Logger
}