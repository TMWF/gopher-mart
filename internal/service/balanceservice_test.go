package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBalanceRepository struct {
	mock.Mock
}

func (m *MockBalanceRepository) GetBalanceForUser(ctx context.Context, userID uuid.UUID) (*model.GetBalanceResponseModel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GetBalanceResponseModel), args.Error(1)
}

func (m *MockBalanceRepository) WithdrawForUserOrder(ctx context.Context, userID uuid.UUID, req *model.WithDrawBalanceRequestModel) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockBalanceRepository) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]model.GetUserWithdrawalsResponseModel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.GetUserWithdrawalsResponseModel), args.Error(1)
}

func TestBalanceService_GetBalanceForUser(t *testing.T) {
	fixedUUID := uuid.New()
	expectedResp := &model.GetBalanceResponseModel{
		Current:   500.5,
		Withdrawn: 100.0,
	}

	tests := []struct {
		name           string
		ctx            context.Context
		setupMock      func(m *MockBalanceRepository)
		expectedResult *model.GetBalanceResponseModel
		expectedErr    error
	}{
		{
			name: "Error: UserID not in context",
			ctx:  context.Background(),
			setupMock: func(m *MockBalanceRepository) {
				// Репозиторий не должен вызываться
			},
			expectedResult: nil,
			expectedErr:    ErrUserNotAuthenticated,
		},
		{
			name: "Error: UserID has wrong type in context",
			ctx:  context.WithValue(context.Background(), util.UserID, "not-a-uuid-type"),
			setupMock: func(m *MockBalanceRepository) {
				// Репозиторий не должен вызываться
			},
			expectedResult: nil,
			expectedErr:    ErrUserNotAuthenticated,
		},
		{
			name: "Error: Repository failure",
			ctx:  context.WithValue(context.Background(), util.UserID, fixedUUID),
			setupMock: func(m *MockBalanceRepository) {
				m.On("GetBalanceForUser", mock.Anything, fixedUUID).
					Return(nil, errors.New("db error"))
			},
			expectedResult: nil,
			expectedErr:    errors.New("db error"),
		},
		{
			name: "Success",
			ctx:  context.WithValue(context.Background(), util.UserID, fixedUUID),
			setupMock: func(m *MockBalanceRepository) {
				m.On("GetBalanceForUser", mock.Anything, fixedUUID).
					Return(expectedResp, nil)
			},
			expectedResult: expectedResp,
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := new(MockBalanceRepository)
			tt.setupMock(repo)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			svc := NewBalanceService(logger, repo)

			// Act
			result, err := svc.GetBalanceForUser(tt.ctx)

			// Assert
			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, ErrUserNotAuthenticated) {
					assert.True(t, errors.Is(err, tt.expectedErr))
				} else {
					assert.Contains(t, err.Error(), tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestBalanceService_WithdrawForUserOrder(t *testing.T) {
	fixedUUID := uuid.New()
	withdrawReq := &model.WithDrawBalanceRequestModel{
		Order: "1234567890",
		Sum:   100.0,
	}

	tests := []struct {
		name        string
		ctx         context.Context
		requestBody *model.WithDrawBalanceRequestModel
		setupMock   func(m *MockBalanceRepository)
		expectedErr error
	}{
		{
			name:        "Error: UserID not in context",
			ctx:         context.Background(),
			requestBody: withdrawReq,
			setupMock: func(m *MockBalanceRepository) {
				// Репозиторий не должен вызываться
			},
			expectedErr: ErrUserNotAuthenticated,
		},
		{
			name:        "Error: UserID has wrong type in context",
			ctx:         context.WithValue(context.Background(), util.UserID, "not-a-uuid-type"),
			requestBody: withdrawReq,
			setupMock: func(m *MockBalanceRepository) {
				// Репозиторий не должен вызываться
			},
			expectedErr: ErrUserNotAuthenticated,
		},
		{
			name:        "Error: Repository failure",
			ctx:         context.WithValue(context.Background(), util.UserID, fixedUUID),
			requestBody: withdrawReq,
			setupMock: func(m *MockBalanceRepository) {
				m.On("WithdrawForUserOrder", mock.Anything, fixedUUID, withdrawReq).
					Return(errors.New("db error"))
			},
			expectedErr: errors.New("db error"),
		},
		{
			name:        "Success withdrawal",
			ctx:         context.WithValue(context.Background(), util.UserID, fixedUUID),
			requestBody: withdrawReq,
			setupMock: func(m *MockBalanceRepository) {
				m.On("WithdrawForUserOrder", mock.Anything, fixedUUID, withdrawReq).
					Return(nil)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := new(MockBalanceRepository)
			tt.setupMock(repo)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			svc := NewBalanceService(logger, repo)

			// Act
			err := svc.WithdrawForUserOrder(tt.ctx, tt.requestBody)

			// Assert
			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, ErrUserNotAuthenticated) {
					assert.True(t, errors.Is(err, tt.expectedErr))
				} else {
					assert.Contains(t, err.Error(), tt.expectedErr.Error()) // Для общих ошибок
				}
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestBalanceService_GetUserWithdrawals(t *testing.T) {
	processedAt, _ := time.Parse(time.RFC3339, "2023-12-10T15:15:45+03:00")
	secondProcessedAt, _ := time.Parse(time.RFC3339, "2023-12-11T15:15:45+03:00")
	fixedUUID := uuid.New()
	mockWithdrawals := []model.GetUserWithdrawalsResponseModel{
		{
			Order:       "12345",
			Sum:         500.0,
			ProcessedAt: processedAt,
		},
		{
			Order:       "67890",
			Sum:         100.0,
			ProcessedAt: secondProcessedAt,
		},
	}

	tests := []struct {
		name           string
		ctx            context.Context
		setupMock      func(m *MockBalanceRepository)
		expectedResult []model.GetUserWithdrawalsResponseModel
		expectedErr    error
	}{
		{
			name: "Error: UserID not in context",
			ctx:  context.Background(),
			setupMock: func(m *MockBalanceRepository) {
				// Репозиторий не вызывается
			},
			expectedResult: nil,
			expectedErr:    ErrUserNotAuthenticated,
		},
		{
			name: "Error: Repository failure",
			ctx:  context.WithValue(context.Background(), util.UserID, fixedUUID),
			setupMock: func(m *MockBalanceRepository) {
				m.On("GetUserWithdrawals", mock.Anything, fixedUUID).
					Return(nil, errors.New("db error"))
			},
			expectedResult: nil,
			expectedErr:    errors.New("db error"),
		},
		{
			name: "Error: No withdrawals found (Empty Slice)",
			ctx:  context.WithValue(context.Background(), util.UserID, fixedUUID),
			setupMock: func(m *MockBalanceRepository) {
				// Возвращаем пустой слайс
				m.On("GetUserWithdrawals", mock.Anything, fixedUUID).
					Return([]model.GetUserWithdrawalsResponseModel{}, nil)
			},
			expectedResult: nil,
			expectedErr:    ErrNotFoundUserWithDrawals,
		},
		{
			name: "Success",
			ctx:  context.WithValue(context.Background(), util.UserID, fixedUUID),
			setupMock: func(m *MockBalanceRepository) {
				m.On("GetUserWithdrawals", mock.Anything, fixedUUID).
					Return(mockWithdrawals, nil)
			},
			expectedResult: mockWithdrawals,
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := new(MockBalanceRepository)
			tt.setupMock(repo)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			svc := NewBalanceService(logger, repo)

			// Act
			result, err := svc.GetUserWithdrawals(tt.ctx)

			// Assert
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedErr) || err.Error() == tt.expectedErr.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
				assert.Len(t, result, 2)
			}

			repo.AssertExpectations(t)
		})
	}
}
