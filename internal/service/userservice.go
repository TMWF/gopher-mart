package service

import (
	"context"
	"log/slog"
)

type UserService interface {
}

type defaultUserService struct {
	log *slog.Logger
}

func NewUserService(log *slog.Logger) *defaultUserService {
	return &defaultUserService{
		log: log.With(slog.String("op", "service.UserService")),
	}
}

func (s *defaultUserService) CreateUser(ctx context.Context, email string) error {
	const op = "CreateUser"
	log := s.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("attempting to create user")

	// if email == "" {
	// 	log.Error("invalid email", logger.Err(ErrInvalidEmail))
	// 	return ErrInvalidEmail
	// }

	log.Info("user created successfully")
	return nil
}
