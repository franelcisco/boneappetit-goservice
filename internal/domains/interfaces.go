package domains

import (
	"bone_appetit_suscription/internal/models"
	"context"
)

type OrderService interface {
	ScheduleNextOrder(ctx context.Context, req models.CreateOrderRequest) error
	ScheduleOrders(ctx context.Context) error
	CreateOnDemandOrderProcess(ctx context.Context, orderNo string) (string, error)
	ReminderProcess(ctx context.Context) error
}

type WebhookService interface {
	ProcessDraftOrderWebhook(ctx context.Context, webhook models.DraftOrderWebhook)
	ProcessPaidOrderWebhook(ctx context.Context, webhook models.OrderWebhook)
}
