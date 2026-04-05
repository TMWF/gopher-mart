package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/service"
	"github.com/TMWF/gopher-mart/internal/util"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService - мок для сервиса пользователя
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, req *model.UserRegisterRequestModel) (*uuid.UUID, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

func (m *MockUserService) LoginUser(ctx context.Context, req *model.UserLoginRequestModel) (*uuid.UUID, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

// MockJWTBuilder - мок для создания JWT
type MockJWTBuilder struct {
	mock.Mock
}

func (m *MockJWTBuilder) BuildJWTString(userID uuid.UUID) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func TestUserHandler_RegisterUser(t *testing.T) {
	// Инициализируем реальный валидатор
	v := validator.New()

	// Константы для тестов
	userID, _ := uuid.NewRandom()
	token := "test-jwt-token"

	tests := []struct {
		name           string
		method         string
		body           interface{}
		setupMock      func(ms *MockUserService, mj *MockJWTBuilder)
		expectedStatus int
		checkCookie    bool
	}{
		{
			name:           "Incorrect Method",
			method:         http.MethodGet,
			body:           model.UserRegisterRequestModel{Login: "test", Password: "password"},
			setupMock:      func(ms *MockUserService, mj *MockJWTBuilder) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Malformed JSON",
			method:         http.MethodPost,
			body:           "invalid json",
			setupMock:      func(ms *MockUserService, mj *MockJWTBuilder) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Validation Failed",
			method:         http.MethodPost,
			body:           model.UserRegisterRequestModel{Login: "", Password: ""}, // Пустые поля
			setupMock:      func(ms *MockUserService, mj *MockJWTBuilder) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "User Already Exists",
			method: http.MethodPost,
			body:   model.UserRegisterRequestModel{Login: "exists", Password: "password"},
			setupMock: func(ms *MockUserService, mj *MockJWTBuilder) {
				ms.On("CreateUser", mock.Anything, mock.Anything).Return(nil, service.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "Success Registration",
			method: http.MethodPost,
			body:   model.UserRegisterRequestModel{Login: "newuser", Password: "password"},
			setupMock: func(ms *MockUserService, mj *MockJWTBuilder) {
				ms.On("CreateUser", mock.Anything, mock.Anything).Return(&userID, nil)
				mj.On("BuildJWTString", userID).Return(token, nil)
			},
			expectedStatus: http.StatusOK,
			checkCookie:    true,
		},
		{
			name:   "JWT Generation Error",
			method: http.MethodPost,
			body:   model.UserRegisterRequestModel{Login: "newuser", Password: "password"},
			setupMock: func(ms *MockUserService, mj *MockJWTBuilder) {
				ms.On("CreateUser", mock.Anything, mock.Anything).Return(&userID, nil)
				mj.On("BuildJWTString", userID).Return("", errors.New("jwt error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ms := new(MockUserService)
			mj := new(MockJWTBuilder)
			tt.setupMock(ms, mj)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			h := NewUserHandler(ms, logger, mj, v)

			var buf bytes.Buffer
			if s, ok := tt.body.(string); ok {
				buf.WriteString(s)
			} else {
				_ = json.NewEncoder(&buf).Encode(tt.body)
			}

			req := httptest.NewRequest(tt.method, "/api/user/register", &buf)
			w := httptest.NewRecorder()

			// Act
			h.RegisterUser(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkCookie {
				cookies := w.Result().Cookies()
				assert.NotEmpty(t, cookies)

				// Проверяем имя куки (util.UserID)
				var found bool
				for _, c := range cookies {
					if c.Name == string(util.UserID) {
						assert.Equal(t, token, c.Value)
						assert.True(t, c.HttpOnly)
						found = true
					}
				}
				assert.True(t, found, "Auth cookie not found")
			}

			ms.AssertExpectations(t)
			mj.AssertExpectations(t)
		})
	}
}

func TestUserHandler_LoginUser(t *testing.T) {
	v := validator.New()
	userID, _ := uuid.NewRandom()
	token := "valid-jwt-token"

	tests := []struct {
		name           string
		method         string
		body           interface{}
		setupMock      func(ms *MockUserService, mj *MockJWTBuilder)
		expectedStatus int
		checkCookie    bool
	}{
		{
			name:           "Method Not Allowed",
			method:         http.MethodGet,
			body:           model.UserLoginRequestModel{Login: "user", Password: "pwd"},
			setupMock:      func(ms *MockUserService, mj *MockJWTBuilder) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			body:           "not-a-json",
			setupMock:      func(ms *MockUserService, mj *MockJWTBuilder) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Validation Failed",
			method:         http.MethodPost,
			body:           model.UserLoginRequestModel{Login: "", Password: ""},
			setupMock:      func(ms *MockUserService, mj *MockJWTBuilder) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Invalid Credentials (User Not Found)",
			method: http.MethodPost,
			body:   model.UserLoginRequestModel{Login: "wrong", Password: "pwd"},
			setupMock: func(ms *MockUserService, mj *MockJWTBuilder) {
				ms.On("LoginUser", mock.Anything, mock.Anything).Return(nil, service.ErrUserDoesNotExist)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Service Error",
			method: http.MethodPost,
			body:   model.UserLoginRequestModel{Login: "user", Password: "pwd"},
			setupMock: func(ms *MockUserService, mj *MockJWTBuilder) {
				ms.On("LoginUser", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "JWT Generation Error",
			method: http.MethodPost,
			body:   model.UserLoginRequestModel{Login: "user", Password: "pwd"},
			setupMock: func(ms *MockUserService, mj *MockJWTBuilder) {
				ms.On("LoginUser", mock.Anything, mock.Anything).Return(&userID, nil)
				mj.On("BuildJWTString", userID).Return("", errors.New("jwt error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "Success Login",
			method: http.MethodPost,
			body:   model.UserLoginRequestModel{Login: "correct", Password: "pwd"},
			setupMock: func(ms *MockUserService, mj *MockJWTBuilder) {
				ms.On("LoginUser", mock.Anything, mock.Anything).Return(&userID, nil)
				mj.On("BuildJWTString", userID).Return(token, nil)
			},
			expectedStatus: http.StatusOK,
			checkCookie:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ms := new(MockUserService)
			mj := new(MockJWTBuilder)
			tt.setupMock(ms, mj)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			h := NewUserHandler(ms, logger, mj, v)

			var buf bytes.Buffer
			if s, ok := tt.body.(string); ok {
				buf.WriteString(s)
			} else {
				_ = json.NewEncoder(&buf).Encode(tt.body)
			}

			req := httptest.NewRequest(tt.method, "/api/user/login", &buf)
			w := httptest.NewRecorder()

			// Act
			h.LoginUser(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkCookie {
				cookies := w.Result().Cookies()
				found := false
				for _, c := range cookies {
					if c.Name == string(util.UserID) {
						assert.Equal(t, token, c.Value)
						assert.True(t, c.HttpOnly)
						assert.Equal(t, 3600*24, c.MaxAge)
						found = true
					}
				}
				assert.True(t, found, "Cookie with JWT not found")
			}

			ms.AssertExpectations(t)
			mj.AssertExpectations(t)
		})
	}
}
