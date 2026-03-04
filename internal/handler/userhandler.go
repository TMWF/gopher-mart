package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/service"
)

type userHandler struct {
	userService service.UserService
	logger      *slog.Logger
}

func NewUserHandler(us service.UserService, logger *slog.Logger) *userHandler {
	return &userHandler{userService: us, logger: logger.With(slog.String("op", "handler.UserHandler"))}
}

func (h *userHandler) RegisterUser(w http.ResponseWriter, req *http.Request) {
	log := h.logger.With(slog.String("op", "handler.RegisterUser"))

	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody model.UseRegisterRequestModel
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqBody); err != nil {
		log.Error("Error occured while decoding request body", logger.Err(err))
		http.Error(w, "Error occured while decoding request body", http.StatusBadRequest)
		return
	}
}

func (h *userHandler) LoginUser(w http.ResponseWriter, req *http.Request) {
	log := h.logger.With(slog.String("op", "LoginUser"))

	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody model.UserLoginRequestModel
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqBody); err != nil {
		log.Error("Error occured while decoding request body", logger.Err(err))
		http.Error(w, "Error occured while decoding request body", http.StatusBadRequest)
		return
	}
}
