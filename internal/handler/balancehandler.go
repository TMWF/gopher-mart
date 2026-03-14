package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/repository"
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

// GetBalanceForUser handles an HTTP GET request to retrieve the current user's balance.
//
// This method extracts user information from the request context,
// requests balance and withdrawal data from the service layer, and returns a JSON response
// containing the current balance and total withdrawn amount.
//
// Responses:
//   - 200 OK: Balance retrieved successfully.
//   - 401 Unauthorized: User is not authenticated.
//   - 405 Method Not Allowed: Incorrect HTTP method used (only GET is allowed).
//   - 500 Internal Server Error: An error occurred while fetching user balance or processing data.
func (bh *balanceHandler) GetBalanceForUser(w http.ResponseWriter, req *http.Request) {
	log := bh.logger.With(slog.String("op", "GetBalanceForUser"))

	if req.Method != http.MethodGet {
		http.Error(w, "Incorrect HTTP method, only GET methods allowed", http.StatusMethodNotAllowed)
		return
	}

	response, err := bh.service.GetBalanceForUser(req.Context())
	if errors.Is(err, service.ErrUserNotAuthenticated) {
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
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

// WithdrawForOrder method is responsible for handling POST-requests for `/api/user/balance/withdraw` endpoint.
// Accepts requst body represented by model.WithDrawBalanceRequestModel struct.

// Responses:
// - 200 OK: when request is successfully handled
// - 401 StatusUnauthorized: when user is not logged in
// - 402 StatusPaymentRequired: when user has not enough bonuses on his balance
// - 422 StatusUnprocessableEntity: when incorrect order is provided in request
// - 500 StatusInternalServerError: for any other error occured while handling request
func (bh *balanceHandler) WithdrawForOrder(w http.ResponseWriter, req *http.Request) {
	log := bh.logger.With(slog.String("op", "WithdrawForOrder"))

	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody model.WithDrawBalanceRequestModel
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&reqBody); err != nil {
		log.Error("Error occured while decoding request body", logger.Err(err))
		http.Error(w, "Error occured while decoding request body", http.StatusBadRequest)
		return
	}

	err := bh.service.WithdrawForUserOrder(req.Context(), &reqBody)
	if errors.Is(err, service.ErrUserNotAuthenticated) {
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if errors.Is(err, repository.ErrBalanceNotEnough) {
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusPaymentRequired)
		return
	}

	if errors.Is(err, repository.ErrIncorrectUserOrder) {
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err != nil {
		log.Error("Error occured while trying to withdraw bonuses for order",
			slog.String("order", reqBody.Order),
			logger.Err(err),
		)
		http.Error(w, "Error occured while trying to withdraw bonuses for order", http.StatusInternalServerError)
		return
	}
}

func (bh *balanceHandler) GetUserWithdrawals(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}
}
