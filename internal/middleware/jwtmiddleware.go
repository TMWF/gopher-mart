package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/TMWF/gopher-mart/internal/config"
	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/util"
	"github.com/golang-jwt/jwt/v4"
)

func JwtTokenMiddleware(config *config.Config, log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jwtCookie, _ := r.Cookie(string(util.UserID))

			if jwtCookie != nil {
				token := jwtCookie.Value
				log.Info("Got token from the cookie",
					slog.String("Encrypted token", token),
				)
				userID, err := getUserID(token, config, log)
				if err != nil {
					log.Error("Error occured while parsing jwtToken", logger.Err(err))
				} else {
					ctx := context.WithValue(r.Context(), util.UserID, userID)
					r = r.WithContext(ctx)
				}
			} else {
				log.Debug("JWT cookie is absent in request")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getUserID(tokenString string, config *config.Config, log *slog.Logger) (int, error) {
	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(config.SecretKey), nil
		})
	if err != nil {
		return -1, err
	}

	if !token.Valid {
		log.Error("Token is not valid")
		return -1, errors.New("token is not valid")
	}

	log.Info("Token is valid")
	return claims.UserID, nil
}
