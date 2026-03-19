package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/TMWF/gopher-mart/internal/model"
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
	// GetUserOrders orchestrates the retrieval of orders for the currently authenticated user.
	//
	// The method follows these steps:
	//  1. Extracts the user's UUID from the provided context using the util.UserID key.
	//  2. Enforces a 5-second processing timeout for the downstream repository call.
	//  3. Fetches the order list from the repository.
	//  4. Validates the result set: if no orders are found, it returns a specific
	//     ErrNotFoundUserOrders error to allow the caller to handle empty states (e.g., 204 No Content).
	//
	// Parameters:
	//   - ctx: The request context, which must contain a valid userID.
	//
	// Returns:
	//   - []model.GetUserOrdersResponseModel: A slice of order data models if found.
	//   - ErrUserNotAuthenticated: If the userID is missing or invalid in the context.
	//   - ErrNotFoundUserOrders: If the repository returns an empty list.
	//   - error: Any other internal error from the repository layer.
	GetUserOrders(ctx context.Context) ([]model.GetUserOrdersResponseModel, error)
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

// Implementation of GetUserOrders of OrdersService interface.
//
// GetUserOrders orchestrates the retrieval of orders for the currently authenticated user.
//
// The method follows these steps:
//  1. Extracts the user's UUID from the provided context using the util.UserID key.
//  2. Enforces a 5-second processing timeout for the downstream repository call.
//  3. Fetches the order list from the repository.
//  4. Validates the result set: if no orders are found, it returns a specific
//     ErrNotFoundUserOrders error to allow the caller to handle empty states (e.g., 204 No Content).
//
// Parameters:
//   - ctx: The request context, which must contain a valid userID.
//
// Returns:
//   - []model.GetUserOrdersResponseModel: A slice of order data models if found.
//   - ErrUserNotAuthenticated: If the userID is missing or invalid in the context.
//   - ErrNotFoundUserOrders: If the repository returns an empty list.
//   - error: Any other internal error from the repository layer.
func (os *ordersService) GetUserOrders(ctx context.Context) ([]model.GetUserOrdersResponseModel, error) {
	log := os.logger.With(slog.String("op", "GetUserOrders"))
	userID, ok := ctx.Value(util.UserID).(uuid.UUID)
	if !ok {
		log.Warn("User id not found in context")
		return nil, ErrUserNotAuthenticated
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := os.repository.GetUserOrders(timeoutCtx, userID)

	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, ErrNotFoundUserOrders
	}

	return result, nil
}
