package util

import (
	"testing"
	"time"

	"github.com/TMWF/gopher-mart/internal/config"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultJWTBuilder_BuildJWTString(t *testing.T) {
	// Arrange
	secret := "test-secret-key"
	expiration := 1 * time.Hour

	cfg := &config.Config{
		SecretKey: secret,
		TokenExp:  expiration,
	}

	builder := NewJWTBuilder(cfg)
	userID := uuid.New()

	// Act
	tokenString, err := builder.BuildJWTString(userID)

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	parsedToken, err := jwt.ParseWithClaims(tokenString, &model.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})

	require.NoError(t, err)
	require.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(*model.Claims)
	require.True(t, ok)

	assert.Equal(t, userID, claims.UserID)

	expectedExp := time.Now().Add(expiration).Unix()
	assert.WithinDuration(t,
		time.Unix(expectedExp, 0),
		claims.ExpiresAt.Time,
		2*time.Second,
	)
}

func TestDefaultJWTBuilder_BuildJWTString_InvalidSecret(t *testing.T) {

	cfg := &config.Config{
		SecretKey: "",
		TokenExp:  -1 * time.Hour,
	}

	builder := NewJWTBuilder(cfg)
	userID := uuid.New()

	tokenString, err := builder.BuildJWTString(userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)
}
