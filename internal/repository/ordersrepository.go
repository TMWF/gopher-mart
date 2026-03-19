package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrdersRepository interface {
	// SaveOrder attempts to persist a new order associated with the given user ID.
	//
	// It uses an atomic SQL query with a Common Table Expression (CTE) to handle
	// potential unique constraint conflicts on the 'order_id' column. This approach
	// prevents race conditions that could occur with a separate "select-then-insert" logic.
	//
	// The method identifies whether the order is new, already exists for the current
	// user, or belongs to another user in a single database round-trip.
	//
	// Parameters:
	//   - ctx: Context for the database operation (supports cancellation and timeouts).
	//   - userID: The UUID of the user attempting to upload the order.
	//   - order: The unique string identifier of the order.
	//
	// Returns:
	//   - nil: If the order was successfully registered.
	//   - ErrOrderAlreadyUploadedByThisUser: If the order already exists and is owned by the same userID.
	//   - ErrOrderAlreadyUploadedByAnotherUser: If the order already exists but is owned by a different user.
	//   - error: A wrapped database error if the query execution or scanning fails.
	SaveOrder(ctx context.Context, userID uuid.UUID, order string) error
	// GetUserOrders retrieves the complete history of orders for a specific user.
	//
	// The method queries the database for order details including ID, status,
	// accrual amount, and creation timestamp. Results are sorted in descending
	// order by 'created_at', ensuring the most recent orders appear first.
	//
	// Parameters:
	//   - ctx: Database context for cancellation and timeouts.
	//   - userID: The unique identifier (UUID) of the user.
	//
	// Returns:
	//   - []model.GetUserOrdersResponseModel: A list of orders (empty slice if none found).
	//   - error: Database execution errors or row scanning failures.
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.GetUserOrdersResponseModel, error)
}

type ordersRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewOrdersRepository initializes and returns a new instance of ordersRepository.
//
// This constructor sets up the data access layer by injecting a PostgreSQL
// connection pool and a structured logger. The logger is pre-configured with
// an "op" (operation) context "repository.OrdersRepository" to facilitate
// precise tracing of database-level diagnostics and errors.
//
// Parameters:
//   - pool: A thread-safe *pgxpool.Pool for managing PostgreSQL connections.
//   - logger: A pointer to an slog.Logger for structured logging.
//
// Returns:
//   - A pointer to the initialized ordersRepository.
func NewOrdersRepository(pool *pgxpool.Pool, logger *slog.Logger) *ordersRepository {
	return &ordersRepository{
		pool:   pool,
		logger: logger.With(slog.String("op", "repository.OrdersRepository")),
	}
}

// Implementation of SaveOrder method of OrdersRepository interface.
// SaveOrder attempts to persist a new order associated with the given user ID.
//
// It uses an atomic SQL query with a Common Table Expression (CTE) to handle
// potential unique constraint conflicts on the 'order_id' column. This approach
// prevents race conditions that could occur with a separate "select-then-insert" logic.
//
// The method identifies whether the order is new, already exists for the current
// user, or belongs to another user in a single database round-trip.
//
// Parameters:
//   - ctx: Context for the database operation (supports cancellation and timeouts).
//   - userID: The UUID of the user attempting to upload the order.
//   - order: The unique string identifier of the order.
//
// Returns:
//   - nil: If the order was successfully registered.
//   - ErrOrderAlreadyUploadedByThisUser: If the order already exists and is owned by the same userID.
//   - ErrOrderAlreadyUploadedByAnotherUser: If the order already exists but is owned by a different user.
//   - error: A wrapped database error if the query execution or scanning fails.
func (or *ordersRepository) SaveOrder(ctx context.Context, userID uuid.UUID, order string) error {
	const query = `
		WITH inserted AS (
		INSERT INTO orders (order_id, user_id, status)
		VALUES ($1, $2, 'NEW')
		ON CONFLICT (order_id) DO NOTHING
		RETURNING user_id
		)
		SELECT user_id, true FROM inserted
		UNION ALL
		SELECT user_id, false FROM orders WHERE order_id = $1;`

	var existingUserID uuid.UUID
	var isNew bool

	err := or.pool.QueryRow(ctx, query, order, userID).Scan(&existingUserID, &isNew)
	if err != nil {
		return fmt.Errorf("failed to upload order: %w", err)
	}

	if isNew {
		return nil
	}

	if existingUserID == userID {
		return ErrOrderAlreadyUploadedByThisUser
	}

	return ErrOrderAlreadyUploadedByAnotherUser
}

// Implementation of GetUserOrders method of OrdersRepository interface.
//
// GetUserOrders retrieves the complete history of orders for a specific user.
//
// The method queries the database for order details including ID, status,
// accrual amount, and creation timestamp. Results are sorted in descending
// order by 'created_at', ensuring the most recent orders appear first.
//
// Parameters:
//   - ctx: Database context for cancellation and timeouts.
//   - userID: The unique identifier (UUID) of the user.
//
// Returns:
//   - []model.GetUserOrdersResponseModel: A list of orders (empty slice if none found).
//   - error: Database execution errors or row scanning failures.
func (or *ordersRepository) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.GetUserOrdersResponseModel, error) {
	query := `SELECT o.order_id, o.status, o.accrual, o.created_at 
	FROM orders as o
	JOIN users as u ON u.id = o.user_id
	WHERE u.id = $1
	ORDER BY o.created_at DESC`

	rows, err := or.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]model.GetUserOrdersResponseModel, 0)

	for rows.Next() {
		var responseModel model.GetUserOrdersResponseModel

		err := rows.Scan(&responseModel.Number, &responseModel.Status, &responseModel.Accrual, &responseModel.UploadedAt)
		if err != nil {
			return nil, err
		}

		result = append(result, responseModel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
