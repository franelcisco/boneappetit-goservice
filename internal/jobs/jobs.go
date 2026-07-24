package jobs

import (
	"context"
	"errors"
	"time"

	"bone_appetit_suscription/internal/domains"
	"bone_appetit_suscription/internal/models"

	"go.uber.org/zap"
)

type JobHandler struct {
	service domains.OrderService
	logger  *zap.Logger
}
type Job struct {
	OrderRequest models.CreateOrderRequest
}

// JobQueue es un canal para enviar trabajos.
var JobQueue chan Job

func NewJobHandler(service domains.OrderService, logger *zap.Logger) *JobHandler {
	return &JobHandler{service: service, logger: logger}
}

// HandleScheduledOrders handles the scheduling of orders
func (h *JobHandler) HandleScheduledOrders() {
	if err := h.service.ScheduleOrders(context.Background()); err != nil {
		h.logger.Error("failed to schedule orders", zap.Error(err))
		return
	}
}

// HandleScheduledOrders handles the scheduling of orders
func (h *JobHandler) HandleReminder() {
	if err := h.service.ReminderProcess(context.Background()); err != nil {
		h.logger.Error("failed to send reminder emails", zap.Error(err))
		return
	}
}

func (h *JobHandler) StartWorkerPool(numWorkers int) {
	JobQueue = make(chan Job, 100) // Cola con buffer de 100 trabajos

	for i := 1; i <= numWorkers; i++ {
		go h.worker(i, JobQueue)
	}
	h.logger.Info("Started workers", zap.Int("num_workers", numWorkers))
}

func (h *JobHandler) worker(id int, jobs <-chan Job) {
	const maxAttempts = 3
	for job := range jobs {
		var err error
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			err = h.service.ScheduleNextOrder(context.Background(), job.OrderRequest)
			if err == nil || !errors.Is(err, domains.OrderNotFound) {
				break
			}
			if attempt < maxAttempts {
				h.logger.Warn("Order not found, retrying", zap.Int("worker_id", id), zap.String("order_id", job.OrderRequest.OrderID), zap.Int("attempt", attempt))
				time.Sleep(1 * time.Second)
			}
		}
		if err != nil {
			h.logger.Error("Worker failed job for order", zap.Int("worker_id", id), zap.String("order_id", job.OrderRequest.OrderID), zap.Error(err))
		}
	}
}
