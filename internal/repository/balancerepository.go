package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBalanceNotEnough = errors.New("not enough bonuses on user balance")
var ErrIncorrectUserOrder = errors.New("incorrect user order")

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

	query := `SELECT b.current_balance, COALESCE(SUM(w.sum), 0) FROM balances as b
	JOIN users as u ON u.id = b.user_id
	JOIN orders as o ON o.user_id = u.id
	JOIN withdrawals as w ON w.order_id = o.id
	WHERE u.id = $1
	GROUB BY(u.id, b.current_balance)`

	err := br.pool.QueryRow(ctx, query, userId).Scan(&response.Current, &response.WithDrawn)

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
	var userBalance float64

	selectUserBalanceQuery := `SELECT b.current_balance FROM balances as b
	JOIN users as u ON u.id = b.user_id
	WHERE u.id = $1`

	tx, err := br.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

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
