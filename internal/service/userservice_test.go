package service

import (
	"context"
	"errors"
	"testing"

	"io"
	"log/slog"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository — ручной мок для репозитория
type mockUserRepository struct {
	CreateUserFunc func(ctx context.Context, req *model.UserRegisterRequestModel, passwordHash []byte) (*uuid.UUID, error)
	LoginUserFunc  func(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, string, error)

	CallsCount int
}

func (m *mockUserRepository) LoginUser(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, string, error) {
	m.CallsCount++
	return m.LoginUserFunc(ctx, req)
}

// CreateUser реализует интерфейс repository.UserRepository
func (m *mockUserRepository) CreateUser(ctx context.Context, req *model.UserRegisterRequestModel, passwordHash []byte) (*uuid.UUID, error) {
	m.CallsCount++
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, req, passwordHash)
	}
	panic("CreateUserFunc is not defined in mock")
}

func TestUserService_CreateUser(t *testing.T) {
	// Общие данные для тестов
	testLogin := "testLogin"
	testPass := "password123"
	fixedID := uuid.New()

	tests := []struct {
		name          string
		req           *model.UserRegisterRequestModel
		setupMock     func(m *mockUserRepository)
		expectedID    *uuid.UUID
		expectedError error
	}{
		{
			name: "Success case",
			req:  &model.UserRegisterRequestModel{Login: testLogin, Password: testPass},
			setupMock: func(m *mockUserRepository) {
				m.CreateUserFunc = func(ctx context.Context, req *model.UserRegisterRequestModel, hash []byte) (*uuid.UUID, error) {
					// Проверяем, что пароль был захеширован корректно
					err := bcrypt.CompareHashAndPassword(hash, []byte(testPass))
					if err != nil {
						return nil, errors.New("hash mismatch")
					}
					return &fixedID, nil
				}
			},
			expectedID:    &fixedID,
			expectedError: nil,
		},
		{
			name: "User already exists",
			req:  &model.UserRegisterRequestModel{Login: testLogin, Password: testPass},
			setupMock: func(m *mockUserRepository) {
				m.CreateUserFunc = func(ctx context.Context, req *model.UserRegisterRequestModel, hash []byte) (*uuid.UUID, error) {
					return nil, repository.ErrUserAlreadyExists
				}
			},
			expectedID:    nil,
			expectedError: ErrUserAlreadyExists,
		},
		{
			name: "Repository error",
			req:  &model.UserRegisterRequestModel{Login: testLogin, Password: testPass},
			setupMock: func(m *mockUserRepository) {
				m.CreateUserFunc = func(ctx context.Context, req *model.UserRegisterRequestModel, hash []byte) (*uuid.UUID, error) {
					return nil, errors.New("db error")
				}
			},
			expectedID:    nil,
			expectedError: errors.New("error occured while trying to create user"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &mockUserRepository{}
			tt.setupMock(mockRepo)

			// Настраиваем логгер в "пустоту"
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			svc := NewUserService(logger, mockRepo)

			// Act
			id, err := svc.CreateUser(context.Background(), tt.req)

			// Assert
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}

			// Проверяем, что репозиторий действительно вызывался 1 раз
			assert.Equal(t, 1, mockRepo.CallsCount)
		})
	}
}

func TestUserService_LoginUser(t *testing.T) {
	// Подготавливаем данные
	correctPassword := "correct_password"
	wrongPassword := "wrong_password"

	// Генерируем реальный хеш для теста успешного входа
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	testUserID := uuid.New()

	tests := []struct {
		name          string
		req           *model.UserLoginRequestModel
		setupMock     func(m *mockUserRepository)
		expectedID    *uuid.UUID
		expectedError error
	}{
		{
			name: "Success login",
			req: &model.UserLoginRequestModel{
				Login:    "testLogin",
				Password: correctPassword,
			},
			setupMock: func(m *mockUserRepository) {
				m.LoginUserFunc = func(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, string, error) {
					return &testUserID, string(hashedPassword), nil
				}
			},
			expectedID:    &testUserID,
			expectedError: nil,
		},
		{
			name: "User not found",
			req: &model.UserLoginRequestModel{
				Login:    "testLogin",
				Password: "any_password",
			},
			setupMock: func(m *mockUserRepository) {
				m.LoginUserFunc = func(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, string, error) {
					return nil, "", repository.ErrUserDoesNotExist
				}
			},
			expectedID:    nil,
			expectedError: ErrUserDoesNotExist,
		},
		{
			name: "Invalid password",
			req: &model.UserLoginRequestModel{
				Login:    "testLogin",
				Password: wrongPassword,
			},
			setupMock: func(m *mockUserRepository) {
				m.LoginUserFunc = func(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, string, error) {
					return &testUserID, string(hashedPassword), nil
				}
			},
			expectedID:    nil,
			expectedError: errors.New("error occured while comparing password hashes"),
		},
		{
			name: "Database error",
			req: &model.UserLoginRequestModel{
				Login:    "testLogin",
				Password: "password123",
			},
			setupMock: func(m *mockUserRepository) {
				m.LoginUserFunc = func(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, string, error) {
					return nil, "", errors.New("connection failed")
				}
			},
			expectedID:    nil,
			expectedError: errors.New("error occured while trying to get user from database"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &mockUserRepository{}
			tt.setupMock(mockRepo)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			svc := NewUserService(logger, mockRepo)

			// Act
			gotID, err := svc.LoginUser(context.Background(), tt.req)

			// Assert
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
				assert.Nil(t, gotID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, gotID)
			}

			assert.Equal(t, 1, mockRepo.CallsCount)
		})
	}
}
