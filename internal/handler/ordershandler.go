package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/TMWF/gopher-mart/internal/repository"
	"github.com/TMWF/gopher-mart/internal/service"
	"github.com/TMWF/gopher-mart/internal/util/validation"
)

type ordersHandler struct {
	logger  *slog.Logger
	service service.OrdersService
}

func NewOrdersHandler(logger *slog.Logger, service service.OrdersService) *ordersHandler {
	return &ordersHandler{
		logger:  logger.With("op", slog.String("op", "handler.OrdersHandler")),
		service: service,
	}
}

// UploadOrder handles the submission of a new order number for the authenticated user.
//
// The method expects a POST request with a plain text body containing a digit-only order number.
// It performs validation using the Luhn algorithm before attempting to persist the data.
//
// Logic flows:
//  1. Verifies that the HTTP method is POST.
//  2. Reads the order number from the request body.
//  3. Validates the order number via Luhn algorithm (returns 422 if invalid).
//  4. Attempts to save the order for the current user via service layer.
//
// HTTP Response Codes:
//   - 202 Accepted: Order number accepted for processing.
//   - 200 OK: Order number has already been uploaded by the current user.
//   - 401 Unauthorized: User is not authenticated.
//   - 405 Method Not Allowed: Request method is not POST.
//   - 409 Conflict: Order number has already been uploaded by another user.
//   - 422 Unprocessable Entity: Order number format is invalid (failed Luhn check).
//   - 500 Internal Server Error: Unexpected server-side failure.
func (oh *ordersHandler) UploadOrder(w http.ResponseWriter, req *http.Request) {
	log := oh.logger.With(slog.String("op", "UploadOrder"))
	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	bodyString := string(bodyBytes)
	if !validation.IsLuhnValid(bodyString) {
		log.Error("Order num has not passed Luhn Validation")
		http.Error(w, "Incorrect order num", http.StatusUnprocessableEntity)
		return
	}

	err = oh.service.SaveOrderForUser(req.Context(), bodyString)

	switch {
	case err == nil:
		w.WriteHeader(http.StatusAccepted)

	case errors.Is(err, service.ErrUserNotAuthenticated):
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)

	case errors.Is(err, repository.ErrOrderAlreadyUploadedByThisUser):
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusOK)

	case errors.Is(err, repository.ErrOrderAlreadyUploadedByAnotherUser):
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusConflict)

	default:
		log.Error(err.Error())
		http.Error(w, "Unexpected error occured while trying to save order", http.StatusInternalServerError)
	}
}

func (oh *ordersHandler) GetUserOrders(w http.ResponseWriter, req *http.Request) {
	// log := oh.logger.With(slog.String("op", "UploadOrder"))
	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}
}
