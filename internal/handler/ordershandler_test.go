package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/TMWF/gopher-mart/internal/service"
	"github.com/stretchr/testify/assert"
)

// MockOrdersService - мок для сервиса
type MockOrdersService struct {
	SaveOrderFunc     func(ctx context.Context, orderID string) error
	GetUserOrdersFunc func(ctx context.Context) ([]model.GetUserOrdersResponseModel, error)
}

func (m *MockOrdersService) SaveOrderForUser(ctx context.Context, orderID string) error {
	return m.SaveOrderFunc(ctx, orderID)
}

func (m *MockOrdersService) GetUserOrders(ctx context.Context) ([]model.GetUserOrdersResponseModel, error) {
	return m.GetUserOrdersFunc(ctx)
}

func TestOrdersHandler_UploadOrder(t *testing.T) {
	// Типичные ошибки для тестов
	errInternal := errors.New("db error")

	tests := []struct {
		name           string
		method         string
		body           string
		mockBehavior   func(m *MockOrdersService)
		expectedStatus int
	}{
		{
			name:           "Method Not Allowed",
			method:         http.MethodGet,
			body:           "123",
			mockBehavior:   func(m *MockOrdersService) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid Luhn Algorithm",
			method:         http.MethodPost,
			body:           "123456781234567", // Невалидный номер
			mockBehavior:   func(m *MockOrdersService) {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "Success Upload",
			method: http.MethodPost,
			body:   "1234567812345670", // Валидный по Луну (пример)
			mockBehavior: func(m *MockOrdersService) {
				m.SaveOrderFunc = func(ctx context.Context, orderID string) error {
					return nil
				}
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:   "User Not Authenticated",
			method: http.MethodPost,
			body:   "1234567812345670",
			mockBehavior: func(m *MockOrdersService) {
				m.SaveOrderFunc = func(ctx context.Context, orderID string) error {
					return service.ErrUserNotAuthenticated
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Already Uploaded By Same User",
			method: http.MethodPost,
			body:   "1234567812345670",
			mockBehavior: func(m *MockOrdersService) {
				m.SaveOrderFunc = func(ctx context.Context, orderID string) error {
					return repository.ErrOrderAlreadyUploadedByThisUser
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Already Uploaded By Another User",
			method: http.MethodPost,
			body:   "1234567812345670",
			mockBehavior: func(m *MockOrdersService) {
				m.SaveOrderFunc = func(ctx context.Context, orderID string) error {
					return repository.ErrOrderAlreadyUploadedByAnotherUser
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "Internal Server Error",
			method: http.MethodPost,
			body:   "1234567812345670",
			mockBehavior: func(m *MockOrdersService) {
				m.SaveOrderFunc = func(ctx context.Context, orderID string) error {
					return errInternal
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockOrdersService{}
			tt.mockBehavior(mockService)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewOrdersHandler(logger, mockService)

			req := httptest.NewRequest(tt.method, "/api/user/orders", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.UploadOrder(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestOrdersHandler_GetUserOrders(t *testing.T) {
	uploadedAt, err := time.Parse(time.RFC3339, "2023-12-10T15:15:45+03:00")
	secondUploadedAt, err := time.Parse(time.RFC3339, "2023-12-11T15:15:45+03:00")
	if err != nil {
		fmt.Println("Ошибка парсинга:", err)
		return
	}

	mockOrders := []model.GetUserOrdersResponseModel{
		{Number: "12345", Status: "PROCESSED", Accrual: 500, UploadedAt: uploadedAt},
		{Number: "67890", Status: "NEW", UploadedAt: secondUploadedAt},
	}

	tests := []struct {
		name           string
		method         string
		mockBehavior   func(m *MockOrdersService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Method Not Allowed",
			method:         http.MethodPost,
			mockBehavior:   func(m *MockOrdersService) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "User Not Authenticated",
			method: http.MethodGet,
			mockBehavior: func(m *MockOrdersService) {
				m.GetUserOrdersFunc = func(ctx context.Context) ([]model.GetUserOrdersResponseModel, error) {
					return nil, service.ErrUserNotAuthenticated
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "No Orders Found",
			method: http.MethodGet,
			mockBehavior: func(m *MockOrdersService) {
				m.GetUserOrdersFunc = func(ctx context.Context) ([]model.GetUserOrdersResponseModel, error) {
					return nil, service.ErrNotFoundUserOrders
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "Internal Server Error",
			method: http.MethodGet,
			mockBehavior: func(m *MockOrdersService) {
				m.GetUserOrdersFunc = func(ctx context.Context) ([]model.GetUserOrdersResponseModel, error) {
					return nil, errors.New("db crash")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "Success",
			method: http.MethodGet,
			mockBehavior: func(m *MockOrdersService) {
				m.GetUserOrdersFunc = func(ctx context.Context) ([]model.GetUserOrdersResponseModel, error) {
					return mockOrders, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"number":"12345","status":"PROCESSED","accrual":500,"uploaded_at":"2023-12-10T15:15:45+03:00"},{"number":"67890","status":"NEW","accrual":0,"uploaded_at":"2023-12-11T15:15:45+03:00"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockOrdersService{}
			tt.mockBehavior(mockService)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			h := NewOrdersHandler(logger, mockService)

			req := httptest.NewRequest(tt.method, "/api/user/orders", nil)
			w := httptest.NewRecorder()

			h.GetUserOrders(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.NotEmpty(t, w.Header().Get("Content-Length"))
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}
