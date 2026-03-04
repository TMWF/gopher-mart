package repository

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface{}

type userRepository struct {
	pool   pgxpool.Pool
	logger *slog.Logger
}

func NewUserRepository(pool pgxpool.Pool, logger *slog.Logger) *userRepository {
	return &userRepository{pool: pool, logger: logger}
}
