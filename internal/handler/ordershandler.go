package handler

import (
	"log/slog"
	"net/http"

	"github.com/TMWF/gopher-mart/internal/service"
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

func (oh *ordersHandler) UploadOrder(w http.ResponseWriter, req *http.Request) {
	// log := oh.logger.With(slog.String("op", "UploadOrder"))
	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (oh *ordersHandler) GetUserOrders(w http.ResponseWriter, req *http.Request) {
	// log := oh.logger.With(slog.String("op", "UploadOrder"))
	if req.Method != http.MethodPost {
		http.Error(w, "Incorrect HTTP method, only POST methods allowed", http.StatusMethodNotAllowed)
		return
	}
}
