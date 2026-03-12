package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/service"
)

type balanceHandler struct {
	logger  *slog.Logger
	service service.BalanceService
}

func NewBalanceHandler(logger *slog.Logger, service service.BalanceService) *balanceHandler {
	return &balanceHandler{
		logger:  logger.With(slog.String("op", "handler.BalanceHandler")),
		service: service,
	}
}

func (bh *balanceHandler) GetBalanceForUser(w http.ResponseWriter, req *http.Request) {
	log := bh.logger.With(slog.String("op", "GetBalanceForUser"))

	if req.Method != http.MethodGet {
		http.Error(w, "Incorrect HTTP method, only GET methods allowed", http.StatusMethodNotAllowed)
		return
	}

	response, err := bh.service.GetBalanceForUser(req.Context())
	if errors.Is(err, service.ErrUserNotAuthenticated) {
		log.Error("User not authenticated")
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if err != nil {
		log.Error("Error occured while getting user balance", logger.Err(err))
		http.Error(w, "Error occured while getting user balance", http.StatusInternalServerError)
		return
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))

	w.WriteHeader(http.StatusOK)
	w.Write(responseBody)
}

func (bh *balanceHandler) WithdrawForOrder(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (bh *balanceHandler) GetUserWithdrawals(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}
}
