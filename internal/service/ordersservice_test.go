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

// MockOrdersRepository - мок для репозитория заказов
type MockOrdersRepository struct {
	mock.Mock
}

// FetchUnprocessedOrders implements [repository.OrdersRepository].
func (m *MockOrdersRepository) FetchUnprocessedOrders(ctx context.Context, batchSize int) ([]model.OrderModel, error) {
	panic("unimplemented")
}

// UpdateOrderAndBalance implements [repository.OrdersRepository].
func (m *MockOrdersRepository) UpdateOrderAndBalance(ctx context.Context, order model.OrderModel, accrualResponse model.AccrualResponseModel) error {
	panic("unimplemented")
}

func (m *MockOrdersRepository) SaveOrder(ctx context.Context, userID uuid.UUID, orderID string) error {
	args := m.Called(ctx, userID, orderID)
	return args.Error(0)
}

func (m *MockOrdersRepository) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]model.GetUserOrdersResponseModel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.GetUserOrdersResponseModel), args.Error(1)
}

func TestOrdersService_SaveOrderForUser(t *testing.T) {
	fixedUUID := uuid.New()
	testOrderID := "12345678903" // Пример валидного номера заказа

	tests := []struct {
		name        string
		ctx         context.Context
		orderID     string
		setupMock   func(m *MockOrdersRepository)
		expectedErr error
	}{
		{
			name:    "Error: UserID not in context",
			ctx:     context.Background(),
			orderID: testOrderID,
			setupMock: func(m *MockOrdersRepository) {
				// Репозиторий не должен вызываться
			},
			expectedErr: ErrUserNotAuthenticated,
		},
		{
			name:    "Error: UserID has wrong type in context",
			ctx:     context.WithValue(context.Background(), util.UserID, "not-a-uuid-type"),
			orderID: testOrderID,
			setupMock: func(m *MockOrdersRepository) {
				// Репозиторий не должен вызываться
			},
			expectedErr: ErrUserNotAuthenticated,
		},
		{
			name:    "Error: Repository failure",
			ctx:     context.WithValue(context.Background(), util.UserID, fixedUUID),
			orderID: testOrderID,
			setupMock: func(m *MockOrdersRepository) {
				m.On("SaveOrder", mock.Anything, fixedUUID, testOrderID).
					Return(errors.New("db error"))
			},
			expectedErr: errors.New("db error"),
		},
		{
			name:    "Success: Order saved",
			ctx:     context.WithValue(context.Background(), util.UserID, fixedUUID),
			orderID: testOrderID,
			setupMock: func(m *MockOrdersRepository) {
				m.On("SaveOrder", mock.Anything, fixedUUID, testOrderID).
					Return(nil)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := new(MockOrdersRepository)
			tt.setupMock(repo)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			svc := NewOrdersService(logger, repo)

			// Act
			err := svc.SaveOrderForUser(tt.ctx, tt.orderID)

			// Assert
			if tt.expectedErr != nil {
				assert.Error(t, err)
				// Для проверки конкретных ошибок (например, ErrUserNotAuthenticated)
				if errors.Is(tt.expectedErr, ErrUserNotAuthenticated) {
					assert.True(t, errors.Is(err, tt.expectedErr))
				} else {
					assert.Contains(t, err.Error(), tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestOrdersService_GetUserOrders(t *testing.T) {
	// Настройка логгера (отправляем в никуда, чтобы не спамить в консоль)
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testUserID := uuid.New()
	now := time.Now().Truncate(time.Second) // Округляем для точности сравнения

	// Тестовые данные согласно вашей структуре
	mockOrders := []model.GetUserOrdersResponseModel{
		{
			Number:     "12345",
			Status:     "PROCESSED",
			Accrual:    500.50,
			UploadedAt: now.Add(-1 * time.Hour),
		},
		{
			Number:     "67890",
			Status:     "NEW",
			Accrual:    0,
			UploadedAt: now,
		},
	}

	tests := []struct {
		name           string
		ctx            context.Context
		setupMock      func(repo *MockOrdersRepository)
		expectedResult []model.GetUserOrdersResponseModel
		expectedErr    error
	}{
		{
			name: "Success: Orders found",
			ctx:  context.WithValue(context.Background(), util.UserID, testUserID),
			setupMock: func(repo *MockOrdersRepository) {
				// Используем mock.Anything, так как внутри метода создается новый context.WithTimeout
				repo.On("GetUserOrders", mock.Anything, testUserID).
					Return(mockOrders, nil)
			},
			expectedResult: mockOrders,
			expectedErr:    nil,
		},
		{
			name: "Error: No user ID in context",
			ctx:  context.Background(),
			setupMock: func(repo *MockOrdersRepository) {
				// Репозиторий не должен вызываться
			},
			expectedResult: nil,
			expectedErr:    ErrUserNotAuthenticated,
		},
		{
			name: "Error: Repository returns error",
			ctx:  context.WithValue(context.Background(), util.UserID, testUserID),
			setupMock: func(repo *MockOrdersRepository) {
				repo.On("GetUserOrders", mock.Anything, testUserID).
					Return(nil, errors.New("database connection error"))
			},
			expectedResult: nil,
			expectedErr:    errors.New("database connection error"),
		},
		{
			name: "Error: No orders found (Empty slice)",
			ctx:  context.WithValue(context.Background(), util.UserID, testUserID),
			setupMock: func(repo *MockOrdersRepository) {
				repo.On("GetUserOrders", mock.Anything, testUserID).
					Return([]model.GetUserOrdersResponseModel{}, nil)
			},
			expectedResult: nil,
			expectedErr:    ErrNotFoundUserOrders,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockOrdersRepository)
			tt.setupMock(mockRepo)

			// Инициализируем сервис
			os := NewOrdersService(discardLogger, mockRepo)

			// Act
			result, err := os.GetUserOrders(tt.ctx)

			// Assert
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedErr) || err.Error() == tt.expectedErr.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)

				// Дополнительная проверка полей структуры
				assert.Equal(t, "12345", result[0].Number)
				assert.Equal(t, 500.50, result[0].Accrual)
				assert.NotEmpty(t, result[0].UploadedAt)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
