package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/TMWF/gopher-mart/internal/service"
	"github.com/TMWF/gopher-mart/internal/util/validation"
	"github.com/go-playground/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBalanceService - ручной мок для интерфейса BalanceService
type MockBalanceService struct {
	GetBalanceFunc           func(ctx context.Context) (*model.GetBalanceResponseModel, error)
	WithdrawForUserOrderFunc func(ctx context.Context, req *model.WithDrawBalanceRequestModel) error
	GetUserWithdrawalsFunc   func(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error)
}

func (m *MockBalanceService) GetBalanceForUser(ctx context.Context) (*model.GetBalanceResponseModel, error) {
	return m.GetBalanceFunc(ctx)
}

func (m *MockBalanceService) WithdrawForUserOrder(ctx context.Context, req *model.WithDrawBalanceRequestModel) error {
	return m.WithdrawForUserOrderFunc(ctx, req)
}

func (m *MockBalanceService) GetUserWithdrawals(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error) {
	return m.GetUserWithdrawalsFunc(ctx)
}

func TestBalanceHandler_GetBalanceForUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	successResponse := &model.GetBalanceResponseModel{
		Current:   500.5,
		Withdrawn: 100.0,
	}

	type fields struct {
		mockService *MockBalanceService
	}
	type args struct {
		method string
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		wantStatus   int
		wantBody     interface{}
		checkHeaders bool
	}{
		{
			name: "Success 200 OK",
			fields: fields{
				mockService: &MockBalanceService{
					GetBalanceFunc: func(ctx context.Context) (*model.GetBalanceResponseModel, error) {
						return successResponse, nil
					},
				},
			},
			args:         args{method: http.MethodGet},
			wantStatus:   http.StatusOK,
			wantBody:     successResponse,
			checkHeaders: true,
		},
		{
			name: "Error 405 Method Not Allowed",
			fields: fields{
				mockService: &MockBalanceService{},
			},
			args:       args{method: http.MethodPost},
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "Error 401 Unauthorized",
			fields: fields{
				mockService: &MockBalanceService{
					GetBalanceFunc: func(ctx context.Context) (*model.GetBalanceResponseModel, error) {
						return nil, service.ErrUserNotAuthenticated
					},
				},
			},
			args:       args{method: http.MethodGet},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "Error 500 Internal Server Error",
			fields: fields{
				mockService: &MockBalanceService{
					GetBalanceFunc: func(ctx context.Context) (*model.GetBalanceResponseModel, error) {
						return nil, errors.New("database failure")
					},
				},
			},
			args:       args{method: http.MethodGet},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := NewBalanceHandler(logger, tt.fields.mockService, nil)

			req := httptest.NewRequest(tt.args.method, "/api/user/balance", nil)
			w := httptest.NewRecorder()

			bh.GetBalanceForUser(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var gotBody model.GetBalanceResponseModel
				err := json.Unmarshal(w.Body.Bytes(), &gotBody)
				require.NoError(t, err)
				assert.Equal(t, tt.wantBody, &gotBody)

				if tt.checkHeaders {
					assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
					assert.Equal(t, strconv.Itoa(w.Body.Len()), w.Header().Get("Content-Length"))
				}
			}
		})
	}
}

func TestBalanceHandler_WithdrawForOrder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v := validator.New()
	err := v.RegisterValidation("luhn", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return validation.IsLuhnValid(value)
	})

	if err != nil {
		fmt.Println("Error occured while adding custom validator")
		return
	}

	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		mockSetup      func(m *MockBalanceService)
		expectedStatus int
	}{
		{
			name:   "Success 200 OK",
			method: http.MethodPost,
			requestBody: model.WithDrawBalanceRequestModel{
				Order: "2377225624", // Валидный номер по Луну
				Sum:   100.5,
			},
			mockSetup: func(m *MockBalanceService) {
				m.WithdrawForUserOrderFunc = func(ctx context.Context, req *model.WithDrawBalanceRequestModel) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Error 405 Method Not Allowed",
			method:         http.MethodGet,
			requestBody:    nil,
			mockSetup:      func(m *MockBalanceService) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Error 400 Bad Request (Invalid JSON)",
			method:         http.MethodPost,
			requestBody:    "invalid-json",
			mockSetup:      func(m *MockBalanceService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Error 422 Unprocessable Entity (Validation Failed)",
			method: http.MethodPost,
			requestBody: model.WithDrawBalanceRequestModel{
				Order: "123",
				Sum:   -10,
			},
			mockSetup:      func(m *MockBalanceService) {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "Error 401 Unauthorized",
			method: http.MethodPost,
			requestBody: model.WithDrawBalanceRequestModel{
				Order: "2377225624",
				Sum:   50,
			},
			mockSetup: func(m *MockBalanceService) {
				m.WithdrawForUserOrderFunc = func(ctx context.Context, req *model.WithDrawBalanceRequestModel) error {
					return service.ErrUserNotAuthenticated
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Error 402 Payment Required (Insufficient Balance)",
			method: http.MethodPost,
			requestBody: model.WithDrawBalanceRequestModel{
				Order: "2377225624",
				Sum:   999999,
			},
			mockSetup: func(m *MockBalanceService) {
				m.WithdrawForUserOrderFunc = func(ctx context.Context, req *model.WithDrawBalanceRequestModel) error {
					return repository.ErrBalanceNotEnough
				}
			},
			expectedStatus: http.StatusPaymentRequired,
		},
		{
			name:   "Error 500 Internal Server Error",
			method: http.MethodPost,
			requestBody: model.WithDrawBalanceRequestModel{
				Order: "2377225624",
				Sum:   10,
			},
			mockSetup: func(m *MockBalanceService) {
				m.WithdrawForUserOrderFunc = func(ctx context.Context, req *model.WithDrawBalanceRequestModel) error {
					return errors.New("unexpected database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBalanceService{}
			tt.mockSetup(mockService)
			bh := NewBalanceHandler(logger, mockService, v)

			var body io.Reader
			if tt.requestBody != nil {
				if s, ok := tt.requestBody.(string); ok {
					body = bytes.NewBufferString(s)
				} else {
					jsonBytes, _ := json.Marshal(tt.requestBody)
					body = bytes.NewBuffer(jsonBytes)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/user/balance/withdraw", body)
			w := httptest.NewRecorder()

			bh.WithdrawForOrder(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestBalanceHandler_GetUserWithdrawals(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	processedAt, err := time.Parse(time.RFC3339, "2023-12-10T15:15:45+03:00")
	if err != nil {
		fmt.Println("Ошибка парсинга:", err)
		return
	}

	mockWithdrawals := []model.GetUserWithdrawalsResponseModel{
		{
			Order:       "2377225624",
			Sum:         500,
			ProcessedAt: processedAt,
		},
	}

	tests := []struct {
		name           string
		method         string
		mockSetup      func(m *MockBalanceService)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "Success 200 OK",
			method: http.MethodGet,
			mockSetup: func(m *MockBalanceService) {
				m.GetUserWithdrawalsFunc = func(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error) {
					return mockWithdrawals, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   mockWithdrawals,
		},
		{
			name:   "Error 405 Method Not Allowed",
			method: http.MethodPost,
			mockSetup: func(m *MockBalanceService) {
				m.GetUserWithdrawalsFunc = nil
			},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "Error 401 Unauthorized",
			method: http.MethodGet,
			mockSetup: func(m *MockBalanceService) {
				m.GetUserWithdrawalsFunc = func(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error) {
					return nil, service.ErrUserNotAuthenticated
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Error 204 No Content",
			method: http.MethodGet,
			mockSetup: func(m *MockBalanceService) {
				m.GetUserWithdrawalsFunc = func(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error) {
					return nil, service.ErrNotFoundUserWithDrawals
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "Error 500 Internal Server Error",
			method: http.MethodGet,
			mockSetup: func(m *MockBalanceService) {
				m.GetUserWithdrawalsFunc = func(ctx context.Context) ([]model.GetUserWithdrawalsResponseModel, error) {
					return nil, errors.New("db error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockBalanceService{}
			tt.mockSetup(mockService)
			bh := NewBalanceHandler(logger, mockService, nil)

			req := httptest.NewRequest(tt.method, "/api/user/withdrawals", nil)
			w := httptest.NewRecorder()

			bh.GetUserWithdrawals(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, strconv.Itoa(w.Body.Len()), w.Header().Get("Content-Length"))

				var gotBody []model.GetUserWithdrawalsResponseModel
				err := json.Unmarshal(w.Body.Bytes(), &gotBody)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, gotBody)
			}
		})
	}
}
