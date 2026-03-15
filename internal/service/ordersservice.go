package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/TMWF/gopher-mart/internal/util"
	"github.com/google/uuid"
)

type OrdersService interface {
	// SaveOrderForUser extracts the user identity from the context and persists a new order.
	//
	// The method expects a valid uuid.UUID to be present in the context under the util.UserID key.
	// It enforces a 5-second timeout for the database operation to ensure system stability.
	//
	// Parameters:
	//   - ctx: The incoming request context, expected to contain the authenticated user's ID.
	//   - order: The validated order number string to be saved.
	//
	// Returns:
	//   - nil: If the order was successfully saved.
	//   - ErrUserNotAuthenticated: If the user ID is missing or invalid in the context.
	//   - repository.ErrOrderAlreadyUploadedByThisUser: If the order already exists for this user.
	//   - repository.ErrOrderAlreadyUploadedByAnotherUser: If the order is already assigned to someone else.
	//   - error: Any other internal error from the repository or context.
	SaveOrderForUser(ctx context.Context, order string) error
}

type ordersService struct {
	logger     *slog.Logger
	repository repository.OrdersRepository
}

// NewOrdersService creates and returns a new instance of ordersService.
//
// It performs dependency injection for the logger and the repository layer.
// The provided logger is enriched with a service-specific "op" (operation)
// context to ensure all downstream logs from this service include the
// "service.OrdersService" attribute for better traceability.
//
// Parameters:
//   - logger: A pointer to an slog.Logger for structured logging.
//   - repository: An implementation of the OrdersRepository interface for database operations.
func NewOrdersService(logger *slog.Logger, repository repository.OrdersRepository) *ordersService {
	return &ordersService{
		logger:     logger.With(slog.String("op", "service.OrdersService")),
		repository: repository,
	}
}

// Implementation of SaveOrderForUser method of OrdersService interface.
// SaveOrderForUser extracts the user identity from the context and persists a new order.
//
// The method expects a valid uuid.UUID to be present in the context under the util.UserID key.
// It enforces a 5-second timeout for the database operation to ensure system stability.
//
// Parameters:
//   - ctx: The incoming request context, expected to contain the authenticated user's ID.
//   - order: The validated order number string to be saved.
//
// Returns:
//   - nil: If the order was successfully saved.
//   - ErrUserNotAuthenticated: If the user ID is missing or invalid in the context.
//   - repository.ErrOrderAlreadyUploadedByThisUser: If the order already exists for this user.
//   - repository.ErrOrderAlreadyUploadedByAnotherUser: If the order is already assigned to someone else.
//   - error: Any other internal error from the repository or context.
func (os *ordersService) SaveOrderForUser(ctx context.Context, order string) error {
	log := os.logger.With(slog.String("op", "SaveOrderForUser"))
	userID, ok := ctx.Value(util.UserID).(uuid.UUID)
	if !ok {
		log.Warn("User id not found in context")
		return ErrUserNotAuthenticated
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := os.repository.SaveOrder(timeoutCtx, userID, order)
	if err != nil {
		return err
	}

	return nil
}
