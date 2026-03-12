package repository

import (
	"context"
	"log/slog"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BalanceRepository defines the interface for managing user balance.
// Implementations are responsible for storing and retrieving balance records.
type BalanceRepository interface {
	// GetBalanceForUser retrieves a user balance from database.
	//
	// Parameters:
	//   - ctx: context.Context used in sql-queries.
	//   - userId: user ID to find balance for.
	//
	// Returns:
	//   - model.GetBalanceResponseModel, containing current user balance and sum of withdrawals for user.
	//   - error, if occured any while retrieving data from database
	GetBalanceForUser(ctx context.Context, userId uuid.UUID) (*model.GetBalanceResponseModel, error)
}

type balanceRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// Constructor func for BalanceRepository
//
// Parameters:
//   - logger: slog.Logger pointer
//   - pool: pgxpool.Pool pointer
//
// Returns:
//   - balanceRepository: implementation of BalanceRepository
func NewBalanceRepository(logger *slog.Logger, pool *pgxpool.Pool) *balanceRepository {
	return &balanceRepository{
		logger: logger.With(slog.String("op", "repository.BalanceRepository")),
		pool:   pool,
	}
}

// Implementation of GetBalanceForUser method of BalanceRepository interface.
//
// Parameters:
//   - ctx: context.Context used in sql-queries.
//   - userId: user ID to find balance for.
//
// Returns:
//   - model.GetBalanceResponseModel, containing current user balance and sum of withdrawals for user.
//   - error, if occured any while retrieving data from database
func (br *balanceRepository) GetBalanceForUser(ctx context.Context, userId uuid.UUID) (*model.GetBalanceResponseModel, error) {
	response := model.GetBalanceResponseModel{}

	query := `SELECT b.current_balance, b.withdrawn FROM balances as b
	JOIN users as u ON u.id = b.user_id
	WHERE u.id = $1`

	err := br.pool.QueryRow(ctx, query, userId).Scan(&response.Current, &response.WithDrawn)

	if err != nil {
		return nil, err
	}

	return &response, nil
}
