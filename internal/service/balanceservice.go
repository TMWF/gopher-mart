package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/TMWF/gopher-mart/internal/util"
	"github.com/google/uuid"
)

var ErrUserNotAuthenticated = errors.New("user not authenticated")

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

	context, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, err := bs.repository.GetBalanceForUser(context, userID)
	if err != nil {
		log.Warn("error occured while trying to retrieve user balance from database")
		return nil, err
	}

	return response, nil
}
