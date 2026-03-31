package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	// WithdrawForUserOrder withdraws bonuses for order from user balance.
	// If user has enough bonuses on balance and correct order is passed,
	// inserts entry in withdrawals table and waithdraws bonuses from user's balance.
	//
	// Parameters:
	//   - ctx: context.Context used in sql-queries.
	//   - userId: user ID to find balance for.
	//   - req: request model, contatining order id and bonuses for withdrawal.
	//
	// Returns:
	//   - error:
	//
	// - ErrBalanceNotEnough - if not enough bonuses on juser's balance.
	// - ErrIncorrectUserOrder - if there is no order with passed order id forr user
	// - any other error if occured any while retrieving data from database
	WithdrawForUserOrder(ctx context.Context, userId uuid.UUID, req *model.WithDrawBalanceRequestModel) error
	// GetUserWithdrawals fetches the complete withdrawal history for a specific user from the database.
	//
	// It performs a complex query joining the 'withdrawals', 'orders', and 'users' tables
	// to retrieve the order ID, the withdrawn amount, and the operation timestamp.
	// The resulting list is sorted by creation date in descending order (newest first).
	//
	// Parameters:
	//   - ctx: The context for managing the database query lifecycle, including timeouts and cancellation.
	//   - userID: The UUID of the user whose withdrawal records are being requested.
	//
	// Returns:
	//   - A slice of model.GetUserWithdrawalsResponseModel containing the historical data.
	//   - An empty slice if no records are found (due to the use of make with length 0).
	//   - An error if the SQL query fails, row scanning encounters a type mismatch,
	//     or the underlying database connection is interrupted.
	//
	// Implementation Note:
	// The function ensures proper resource management by deferring rows.Close()
	// and performs a final check for errors encountered during row iteration via rows.Err().
	GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.GetUserWithdrawalsResponseModel, error)
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
func (br *balanceRepository) GetBalanceForUser(ctx context.Context, userID uuid.UUID) (*model.GetBalanceResponseModel, error) {
	response := model.GetBalanceResponseModel{}
	log := br.logger.With(slog.String("op", "GetBalanceForUser"))
	log.Debug("Trying to get balance for user", slog.String("user", userID.String()))

	query := `SELECT b.current_balance, COALESCE(SUM(w.sum), 0) FROM balances as b
	JOIN users as u ON u.id = b.user_id
	LEFT JOIN orders as o ON o.user_id = u.id
	LEFT JOIN withdrawals as w ON w.order_id = o.id
	WHERE u.id = $1
	GROUP BY(u.id, b.current_balance)`

	err := br.pool.QueryRow(ctx, query, userID).Scan(&response.Current, &response.WithDrawn)

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// Implementation of WithdrawForUserOrder method of BalanceRepository interface.
// WithdrawForUserOrder withdraws bonuses for order from user balance
// If user has enough bonuses on balance and correct order is passed,
// inserts entry in withdrawals table and waithdraws bonuses from user's balance.
//
// Parameters:
//   - ctx: context.Context used in sql-queries.
//   - userId: user ID to find balance for.
//   - req: request model, contatining order id and bonuses for withdrawal.
//
// Returns:
//   - error:
//
// - ErrBalanceNotEnough - if not enough bonuses on juser's balance.
// - ErrIncorrectUserOrder - if there is no order with passed order id forr user
// - any other error if occured any while retrieving data from database
func (br *balanceRepository) WithdrawForUserOrder(ctx context.Context, userId uuid.UUID, req *model.WithDrawBalanceRequestModel) error {
	log := br.logger.With(slog.String("op", "WithdrawForUserOrder"))
	var userBalance float64

	selectUserBalanceQuery := `SELECT b.current_balance FROM balances as b
	JOIN users as u ON u.id = b.user_id
	WHERE u.id = $1`

	tx, err := br.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			log.Error(
				"Failed to properly close transaction",
				logger.Err(err),
			)
		}
		log.Debug("Successfully closed transaction")
	}()

	err = tx.QueryRow(ctx, selectUserBalanceQuery, userId).Scan(&userBalance)
	if err != nil {
		return err
	}

	if req.Sum < userBalance {
		return ErrBalanceNotEnough
	}

	var orderId uuid.UUID
	selectOrderIdQuery := `SELECT o.id FROM orders as o
	JOIN users as u ON u.id = o.user_id
	WHERE u.id = $1
	AND o.order_id = $2`

	err = tx.QueryRow(ctx, selectOrderIdQuery, userId, req.Order).Scan(&orderId)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrIncorrectUserOrder
	}

	if err != nil {
		return err
	}

	insertIntoWithDrawalsTable := `INSERT INTO withdrawals (sum, user_id, order_id) 
	VALUES ($1, $2, $3)`

	_, err = tx.Exec(ctx, insertIntoWithDrawalsTable, req.Sum, userId, orderId)
	if err != nil {
		return err
	}

	balanceAfterWithDrawal := userBalance - req.Sum
	updateUserBalanceQuery := `UPDATE balances
	SET balance = $1
	WHERE user_id = $2`

	_, err = tx.Exec(ctx, updateUserBalanceQuery, balanceAfterWithDrawal, userId)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Implemetation of GetUserWithdrawals method of BalanceRepository.
// GetUserWithdrawals fetches the complete withdrawal history for a specific user from the database.
//
// It performs a complex query joining the 'withdrawals', 'orders', and 'users' tables
// to retrieve the order ID, the withdrawn amount, and the operation timestamp.
// The resulting list is sorted by creation date in descending order (newest first).
//
// Parameters:
//   - ctx: The context for managing the database query lifecycle, including timeouts and cancellation.
//   - userID: The UUID of the user whose withdrawal records are being requested.
//
// Returns:
//   - A slice of model.GetUserWithdrawalsResponseModel containing the historical data.
//   - An empty slice if no records are found (due to the use of make with length 0).
//   - An error if the SQL query fails, row scanning encounters a type mismatch,
//     or the underlying database connection is interrupted.
//
// Implementation Note:
// The function ensures proper resource management by deferring rows.Close()
// and performs a final check for errors encountered during row iteration via rows.Err().
func (br *balanceRepository) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.GetUserWithdrawalsResponseModel, error) {
	query := `SELECT o.order_id, w.sum, w.created_at FROM withdrawals as w
	JOIN orders as o ON o.id = w.order_id
	JOIN users as u ON u.id = o.user_id
	WHERE u.id = $1
	ORDER BY w.created_at DESC`

	rows, err := br.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]model.GetUserWithdrawalsResponseModel, 0)

	for rows.Next() {
		var responseModel model.GetUserWithdrawalsResponseModel

		if err := rows.Scan(&responseModel.Order, &responseModel.Sum, &responseModel.ProcessedAt); err != nil {
			return nil, err
		}

		result = append(result, responseModel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
