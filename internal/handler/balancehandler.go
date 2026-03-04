package handler

import (
	"log/slog"
	"net/http"

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
	if req.Method != http.MethodGet {
		http.Error(w, "Incorrect HTTP method, only GET methods allowed", http.StatusMethodNotAllowed)
		return
	}
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
