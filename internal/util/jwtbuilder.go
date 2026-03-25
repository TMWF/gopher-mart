package util

import (
	"time"

	"github.com/TMWF/gopher-mart/internal/config"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type UserJWTBuilder interface {
	BuildJWTString(userID uuid.UUID) (string, error)
}

type defaultJWTBuilder struct {
	config *config.Config
}

func NewJWTBuilder(cfg *config.Config) *defaultJWTBuilder {
	return &defaultJWTBuilder{config: cfg}
}

func (helper *defaultJWTBuilder) BuildJWTString(userID uuid.UUID) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(helper.config.TokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(helper.config.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
