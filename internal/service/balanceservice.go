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

// BalanceService defines the interface for service layer of balancehandler.
// Implementations are responsible for processing requests from balancehandler.
type BalanceService interface {
	// GetBalanceForUser processes get-request for user balance from balancehandler
	// The method checks user authentication and returns ErrUserNotAuthenticated, if user is not authorized
	//
	// Parameters:
	//   - ctx: context.Context used in sql-queries.
	//
	// Returns:
	//   - model.GetBalanceResponseModel, containing current user balance and sum of withdrawals for user.
	//   - error, if user is not authenticated or any error occured while retrieving data from database
	GetBalanceForUser(context.Context) (*model.GetBalanceResponseModel, error)
	// WithdrawForUserOrder processes POST-request from balancehandler for withdrawing bonuses from user's balance
	// The method also checks user authentication and returns ErrUserNotAuthenticated, if user is not authorized
	//
	// Parameters:
	//   - ctx: context.Context used in sql-queries.
	//   - req: request model, passed down from balancehandler method
	//
	// Returns:
	//   - error, if user is not authenticated or any error occured while retrieving data from database
	WithdrawForUserOrder(ctx context.Context, req *model.WithDrawBalanceRequestModel) error
	// GetUserWithdrawals retrieves the withdrawal history for the authenticated user.
	// It extracts the user ID from the context, sets a timeout for the database operation,
	// and calls the repository to fetch withdrawal records.
	//
	// It returns a slice of GetUserWithdrawalsResponseModel if withdrawals are found,
	// or an error in the following cases:
	// - ErrUserNotAuthenticated: If the user ID is not found in the context.
	// - ErrNotFoundUserWithDrawals: If the user has no withdrawal records.
	// - Any other error returned by the repository layer during the database operation.
	//
	// Context:
	// The provided context is used for cancellation and with a timeout of 5 seconds for the repository call.
	// The UserID is expected to be stored in the context under the util.UserID key.
	//
	// Logging:
	// The method logs warnings for missing user ID in context and for cases where no withdrawals are found for a user.
	GetUserWithdrawals(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error)
}

type balanceService struct {
	logger     *slog.Logger
	repository repository.BalanceRepository
}

// Constructor func for BalanceService
//
// Parameters:
//   - logger: slog.Logger pointer
//   - repository: repository.BalanceRepository
//
// Returns:
//   - balanceService: implementation of BalanceService
func NewBalanceService(logger *slog.Logger, repository repository.BalanceRepository) *balanceService {
	return &balanceService{
		logger:     logger.With(slog.String("op", "service.BalanceService")),
		repository: repository,
	}
}

// Implementation of GetBalanceForUser method of BalanceService interface
// GetBalanceForUser processes get-request for user balance from balancehandler
// The method checks user authentication and returns ErrUserNotAuthenticated, if user is not authorized
//
// Parameters:
//   - ctx: context.Context used in sql-queries.
//
// Returns:
//   - model.GetBalanceResponseModel, containing current user balance and sum of withdrawals for user.
//   - error, if user is not authenticated or any error occured while retrieving data from database
func (bs *balanceService) GetBalanceForUser(ctx context.Context) (*model.GetBalanceResponseModel, error) {
	log := bs.logger.With(slog.String("op", "GetBalanceForUser"))
	userID, ok := ctx.Value(util.UserID).(uuid.UUID)
	if !ok {
		log.Warn("User id not found in context")
		return nil, ErrUserNotAuthenticated
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, err := bs.repository.GetBalanceForUser(timeoutCtx, userID)
	if err != nil {
		log.Warn("error occured while trying to retrieve user balance from database")
		return nil, err
	}

	return response, nil
}

// Implementation of WithdrawForUserOrder method of BalanceService interface
// WithdrawForUserOrder processes POST-request from balancehandler for withdrawing bonuses from user's balance
// The method also checks user authentication and returns ErrUserNotAuthenticated, if user is not authorized
//
// Parameters:
//   - ctx: context.Context used in sql-queries.
//   - req: request model, passed down from balancehandler method
//
// Returns:
//   - error, if user is not authenticated or any error occured while retrieving data from database
func (bs *balanceService) WithdrawForUserOrder(ctx context.Context, req *model.WithDrawBalanceRequestModel) error {
	log := bs.logger.With(slog.String("op", "WithdrawForUserOrder"))
	userID, ok := ctx.Value(util.UserID).(uuid.UUID)
	if !ok {
		log.Warn("User id not found in context")
		return ErrUserNotAuthenticated
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return bs.repository.WithdrawForUserOrder(timeoutCtx, userID, req)
}

// Implementation of GetUserWithdrawals method of BalanceService method.
// GetUserWithdrawals retrieves the withdrawal history for the authenticated user.
// It extracts the user ID from the context, sets a timeout for the database operation,
// and calls the repository to fetch withdrawal records.
//
// It returns a slice of GetUserWithdrawalsResponseModel if withdrawals are found,
// or an error in the following cases:
// - ErrUserNotAuthenticated: If the user ID is not found in the context.
// - ErrNotFoundUserWithDrawals: If the user has no withdrawal records.
// - Any other error returned by the repository layer during the database operation.
//
// Context:
// The provided context is used for cancellation and with a timeout of 5 seconds for the repository call.
// The UserID is expected to be stored in the context under the util.UserID key.
//
// Logging:
// The method logs warnings for missing user ID in context and for cases where no withdrawals are found for a user.
func (bs *balanceService) GetUserWithdrawals(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error) {
	log := bs.logger.With(slog.String("op", "GetUserWithdrawals"))
	userID, ok := ctx.Value(util.UserID).(uuid.UUID)
	if !ok {
		log.Warn("User id not found in context")
		return nil, ErrUserNotAuthenticated
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, err := bs.repository.GetUserWithdrawals(timeoutCtx, userID)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 {
		log.Warn("Not found withdrawals for user", slog.String("userID", userID.String()))
		return nil, ErrNotFoundUserWithDrawals
	}

	return response, nil
}
