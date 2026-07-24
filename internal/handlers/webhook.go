package handlers

import (
	"bone_appetit_suscription/internal/domains"
	"bone_appetit_suscription/internal/models"
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WebhookHandler handles incoming webhook requests from Shopify.
type WebhookHandler struct {
	WebhookService domains.WebhookService
	OrderService   domains.OrderService
	logger         *zap.Logger
}

// NewWebhookHandler creates a new instance of WebhookHandler.
func NewWebhookHandler(webhookService domains.WebhookService, orderService domains.OrderService, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		WebhookService: webhookService,
		OrderService:   orderService,
		logger:         logger,
	}
}

// HandleWebhookDraft handles incoming webhook requests from Shopify.
func (h *WebhookHandler) HandleWebhookDraft(c *gin.Context) {

	var req models.DraftOrderWebhook
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error unmarshaling webhook payload: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	go h.WebhookService.ProcessDraftOrderWebhook(context.Background(), req)

	c.JSON(200, gin.H{"status": "success"})
}

// HandleWebhookDraft handles incoming webhook requests from Shopify.
func (h *WebhookHandler) HandleWebhookOrderPaid(c *gin.Context) {
	var req models.OrderWebhook
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Error unmarshaling webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	go h.WebhookService.ProcessPaidOrderWebhook(context.Background(), req)

	c.JSON(200, gin.H{"status": "success"})
}
