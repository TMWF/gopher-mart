package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/TMWF/gopher-mart/internal/logger"
	"github.com/TMWF/gopher-mart/internal/model"
	"github.com/TMWF/gopher-mart/internal/repository"
)

type accrualWorker struct {
	repository   repository.OrdersRepository
	client       *http.Client
	accrualURL   string
	logger       *slog.Logger
	maxWorkers   int
	pollInterval time.Duration
	retryAfter   time.Time
	retryMu      sync.RWMutex
}

func NewAccrualWorker(
	repository repository.OrdersRepository,
	apiHost string,
	logger *slog.Logger,
	maxWorkers int,
	pollInterval time.Duration,
) *accrualWorker {
	return &accrualWorker{
		repository:   repository,
		client:       &http.Client{Timeout: 10 * time.Second},
		accrualURL:   apiHost + "/api/orders/",
		logger:       logger.With(slog.String("op", "worker.AccrualWorker")),
		maxWorkers:   maxWorkers,
		pollInterval: pollInterval * time.Second,
	}
}

func (w *accrualWorker) Run(ctx context.Context) {
	log := w.logger.With(slog.String("op", "Run"))
	orderChan := make(chan model.OrderModel, w.maxWorkers)
	var wg sync.WaitGroup

	for i := 0; i < w.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.worker(ctx, orderChan)
		}()
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		log.Debug("Starting scheduled work")
		select {
		case <-ctx.Done():
			log.Warn("Finishing scheduler")
			close(orderChan)
			wg.Wait()
			return
		case <-ticker.C:
			w.retryMu.RLock()
			waiting := time.Now().Before(w.retryAfter)
			w.retryMu.RUnlock()
			if waiting {
				log.Debug("Skipping worker iteration cause still waiting")
				continue
			}

			orders, err := w.repository.FetchUnprocessedOrders(ctx, w.maxWorkers*2)
			if err != nil {
				log.Error("failed to fetch orders", "err", err)
				continue
			}

			for _, order := range orders {
				log.Debug("Sending order to chsnnel",
					slog.String("orderId", order.ID.String()),
					slog.String("orderStatus", order.Status))
				orderChan <- order
			}
		}
	}
}

func (w *accrualWorker) worker(ctx context.Context, orders <-chan model.OrderModel) {
	log := w.logger.With(slog.String("op", "worker"))
	for order := range orders {
		err := w.processOrder(ctx, order)
		if err != nil {
			log.Error("failed to process order", "order_id", order.ID, "err", err)
		}
	}
}

func (w *accrualWorker) processOrder(ctx context.Context, order model.OrderModel) error {
	log := w.logger.With(slog.String("op", "processOrder"))
	w.retryMu.RLock()
	pauseUntil := w.retryAfter
	w.retryMu.RUnlock()

	if time.Now().Before(pauseUntil) {
		return nil
	}

	url := fmt.Sprintf("%s%s", w.accrualURL, order.Num)
	log.Debug("Accrual URL: " + url)
	resp, err := w.client.Get(url)
	if err != nil {
		log.Error("Failed to send request to accrual system", logger.Err(err))
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error(
				"Failed to properly close response body",
				logger.Err(err),
			)
		}
		log.Debug("Successfully closed response body")
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		w.handle429(resp.Header.Get("Retry-After"))
		return fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode == http.StatusNoContent {
		return fmt.Errorf("order not registered in accrual system")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected error (%d) occured while trying to get accrual for order:%s", resp.StatusCode, order.ID)
	}

	var accrualResponse model.AccrualResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&accrualResponse); err != nil {
		return err
	}

	return w.repository.UpdateOrderAndBalance(ctx, order, accrualResponse)
}

func (w *accrualWorker) handle429(retryAfterHeader string) {
	seconds, err := strconv.Atoi(retryAfterHeader)
	if err != nil {
		seconds = 60
	}

	w.retryMu.Lock()
	w.retryAfter = time.Now().Add(time.Duration(seconds) * time.Second)
	w.retryMu.Unlock()

	w.logger.Warn("429 Too Many Requests received", "wait_seconds", seconds)
}
