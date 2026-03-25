package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	// CreateUser attempts to register a new user with a unique login and hashed password.
	//
	// The method performs an atomic INSERT operation. It uses "ON CONFLICT DO NOTHING"
	// to handle login collisions gracefully. If the login is already taken, the query
	// returns no rows, which this method translates into a specific ErrUserAlreadyExists error.
	//
	// To ensure case-insensitive uniqueness, the login is converted to lowercase
	// before being persisted.
	//
	// Parameters:
	//   - ctx: Context for the database operation, supporting cancellation and timeouts.
	//   - req: Request model containing the user's login.
	//   - hashedPassword: The securely hashed password as a byte slice.
	//
	// Returns:
	//   - *uuid.UUID: A pointer to the newly generated user ID on success.
	//   - ErrUserAlreadyExists: If a user with the same login already exists in the system.
	//   - error: A wrapped database error if the query execution or scanning fails.
	CreateUser(ctx context.Context, req *model.UserRegisterRequestModel, hashedPassword []byte) (*uuid.UUID, error)
}

type userRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewUserRepository initializes and returns a new instance of userRepository.
//
// This constructor follows the dependency injection pattern, requiring a
// PostgreSQL connection pool for database operations and a structured logger
// for diagnostics and error reporting.
//
// Parameters:
//   - pool: A thread-safe *pgxpool.Pool instance for managing database connections.
//   - logger: A pointer to a slog.Logger for structured logging.
//
// Returns:
//   - A pointer to the initialized userRepository, ready for data persistence tasks.
func NewUserRepository(pool *pgxpool.Pool, logger *slog.Logger) *userRepository {
	return &userRepository{pool: pool, logger: logger}
}

// Implementation of CreateUser method of UserRepository interface.
//
// CreateUser attempts to register a new user with a unique login and hashed password.
//
// The method performs an atomic INSERT operation. It uses "ON CONFLICT DO NOTHING"
// to handle login collisions gracefully. If the login is already taken, the query
// returns no rows, which this method translates into a specific ErrUserAlreadyExists error.
//
// To ensure case-insensitive uniqueness, the login is converted to lowercase
// before being persisted.
//
// Parameters:
//   - ctx: Context for the database operation, supporting cancellation and timeouts.
//   - req: Request model containing the user's login.
//   - hashedPassword: The securely hashed password as a byte slice.
//
// Returns:
//   - *uuid.UUID: A pointer to the newly generated user ID on success.
//   - ErrUserAlreadyExists: If a user with the same login already exists in the system.
//   - error: A wrapped database error if the query execution or scanning fails.
func (ur *userRepository) CreateUser(ctx context.Context, req *model.UserRegisterRequestModel, hashedPassword []byte) (*uuid.UUID, error) {
	const query = `
	INSERT INTO users (login, password)
	VALUES ($1, $2)
	ON CONFLICT (login) DO NOTHING
	RETURNING id;
	`

	var userID uuid.UUID

	err := ur.pool.QueryRow(ctx, query, strings.ToLower(req.Login), hashedPassword).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserAlreadyExists
		}

		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return &userID, nil
}
