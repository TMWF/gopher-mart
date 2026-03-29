package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	// CreateUser handles the business logic for registering a new user.
	//
	// The process involves:
	//  1. Hashing the plain-text password using the bcrypt algorithm with DefaultCost.
	//  2. Setting a 5-second execution timeout for the database operation.
	//  3. Attempting to persist the user via the repository layer.
	//  4. Mapping repository-specific errors (like ErrUserAlreadyExists) into
	//     service-level domain errors to decouple layers.
	//
	// Parameters:
	//   - ctx: The incoming request context.
	//   - req: The registration model containing user credentials (login and password).
	//
	// Returns:
	//   - *uuid.UUID: A pointer to the unique identifier of the newly created user.
	//   - ErrUserAlreadyExists: If the chosen login is already taken.
	//   - error: If password hashing fails or an unexpected database error occurs.
	CreateUser(ctx context.Context, req *model.UserRegisterRequestModel) (*uuid.UUID, error)
	// LoginUser authenticates a user by verifying their credentials against the stored records.
	//
	// The method performs the following steps:
	//  1. Sets a hard 5-second timeout for the database operation.
	//  2. Retrieves the user's ID and hashed password from the repository.
	//  3. Compares the provided plain-text password with the stored hash using bcrypt.
	//
	// If the user is not found in the database, it returns [ErrUserDoesNotExist].
	// If the password does not match or a database error occurs, it returns a wrapped error.
	LoginUser(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, error)
}

type defaultUserService struct {
	repository repository.UserRepository
	log        *slog.Logger
}

// NewUserService initializes and returns a new instance of the defaultUserService implementation.
//
// This constructor implements the dependency injection pattern, requiring a
// repository.UserRepository interface to decouple business logic from the specific
// data storage implementation.
//
// The provided logger is enriched with a service-level "op" (operation) context
// ("service.UserService") to ensure consistent structured logging across all
// user-related business operations.
//
// Parameters:
//   - log: A pointer to an slog.Logger for structured diagnostic and business logging.
//   - repository: An implementation of the UserRepository interface for data persistence.
//
// Returns:
//   - A pointer to the initialized defaultUserService.
func NewUserService(log *slog.Logger, repository repository.UserRepository) *defaultUserService {
	return &defaultUserService{
		log:        log.With(slog.String("op", "service.UserService")),
		repository: repository,
	}
}

// Implementation of CreateUser interface of UserService interface.
//
// CreateUser handles the business logic for registering a new user.
//
// The process involves:
//  1. Hashing the plain-text password using the bcrypt algorithm with DefaultCost.
//  2. Setting a 5-second execution timeout for the database operation.
//  3. Attempting to persist the user via the repository layer.
//  4. Mapping repository-specific errors (like ErrUserAlreadyExists) into
//     service-level domain errors to decouple layers.
//
// Parameters:
//   - ctx: The incoming request context.
//   - req: The registration model containing user credentials (login and password).
//
// Returns:
//   - *uuid.UUID: A pointer to the unique identifier of the newly created user.
//   - ErrUserAlreadyExists: If the chosen login is already taken.
//   - error: If password hashing fails or an unexpected database error occurs.
func (s *defaultUserService) CreateUser(ctx context.Context, req *model.UserRegisterRequestModel) (*uuid.UUID, error) {
	log := s.log.With(
		slog.String("op", "CreateUser"),
	)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to get hashed password: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	userID, err := s.repository.CreateUser(timeoutCtx, req, hashedPassword)
	if errors.Is(err, repository.ErrUserAlreadyExists) {
		return nil, ErrUserAlreadyExists
	}

	if err != nil {
		return nil, fmt.Errorf("error occured while trying to create user: %w", err)
	}

	log.Info("user created successfully")
	return userID, nil
}

// Implementation of LoginUser method of UserService interface.
//
// LoginUser authenticates a user by verifying their credentials against the stored records.
//
// The method performs the following steps:
//  1. Sets a hard 5-second timeout for the database operation.
//  2. Retrieves the user's ID and hashed password from the repository.
//  3. Compares the provided plain-text password with the stored hash using bcrypt.
//
// If the user is not found in the database, it returns [ErrUserDoesNotExist].
// If the password does not match or a database error occurs, it returns a wrapped error.
func (s *defaultUserService) LoginUser(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	userID, hashedPassword, err := s.repository.LoginUser(timeoutCtx, req)
	if err != nil {
		if errors.Is(err, repository.ErrUserDoesNotExist) {
			return nil, ErrUserDoesNotExist
		}

		return nil, fmt.Errorf("error occured while trying to get user from database: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("error occured while comparing password hashes: %w", err)
	}

	return userID, nil
}
