package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

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

// GetUserOrders handles the retrieval of all orders associated with the authenticated user.
//
// The method expects a GET request and fetches the list of orders from the service layer.
// Results are returned as a JSON-encoded array in the response body. If the user has
// no orders, the server responds with a 204 No Content status.
//
// Logic flows:
//  1. Verifies that the HTTP method is GET.
//  2. Calls the service layer to fetch orders for the user identified in the context.
//  3. Handles authentication errors (401) and empty result cases (204).
//  4. Marshals the list of orders into JSON and sets appropriate response headers.
//
// HTTP Response Codes:
//   - 200 OK: Successfully retrieved the list of orders (returned as JSON).
//   - 204 No Content: The user has no orders registered in the system.
//   - 401 Unauthorized: User identity could not be verified from the context.
//   - 405 Method Not Allowed: The request method is not GET.
//   - 500 Internal Server Error: Unexpected failure during data retrieval or JSON serialization.
func (oh *ordersHandler) GetUserOrders(w http.ResponseWriter, req *http.Request) {
	log := oh.logger.With(slog.String("op", "GetUserOrders"))
	if req.Method != http.MethodGet {
		http.Error(w, "Incorrect HTTP method, only GET methods allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := oh.service.GetUserOrders(req.Context())

	if errors.Is(err, service.ErrUserNotAuthenticated) {
		log.Error(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if errors.Is(err, service.ErrNotFoundUserOrders) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err != nil {
		log.Error(err.Error())
		http.Error(w, "Unexpected error occured while trying to get user orders", http.StatusInternalServerError)
		return
	}

	responseBody, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))

	w.WriteHeader(http.StatusOK)
	w.Write(responseBody)
}
