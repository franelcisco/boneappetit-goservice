package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"bone_appetit_suscription/internal/domains"
	"bone_appetit_suscription/internal/jobs"
	"bone_appetit_suscription/internal/models"
	helpers "bone_appetit_suscription/pkg"
	"bone_appetit_suscription/pkg/db"
	dbModels "bone_appetit_suscription/pkg/db/models"
	"bone_appetit_suscription/pkg/shopify"
)

type webhookService struct {
	DB                *gorm.DB
	Logger            *zap.Logger
	shopifyRepository shopify.Repository
	location          *time.Location
}

func NewWebhookService(db *gorm.DB, logger *zap.Logger, shopifyRepository shopify.Repository, location *time.Location) domains.WebhookService {
	return &webhookService{
		DB:                db,
		Logger:            logger,
		shopifyRepository: shopifyRepository,
		location:          location,
	}
}

func (s *webhookService) ProcessDraftOrderWebhook(ctx context.Context, webhook models.DraftOrderWebhook) {
	if webhook.Status != "completed" {
		return
	}

	currentDate := time.Now().In(s.location)
	// 1. find order in db by last created_at
	var dbOrder dbModels.Order
	err := s.DB.WithContext(ctx).Model(&dbModels.Order{}).
		Where("shopify_draft_order_id = ?", webhook.Name).
		Where("status = ?", "Triggered").
		// Where("customer_email IN ?", testUserEmails).
		First(&dbOrder).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		s.Logger.Error("Error finding order", zap.Error(err))
		return
	}

	if err == gorm.ErrRecordNotFound {
		return
	}

	// 2. get Order from shopify
	resp, err := s.shopifyRepository.GetDraftOrderByID(ctx, fmt.Sprintf("%d", webhook.ID))
	if err != nil {
		return
	}

	if resp.DraftOrder.Order == nil {
		return
	}

	log := models.Log{
		UpdatedField:  "schedule_order_completed",
		NewValue:      resp.DraftOrder.Order.Name,
		PreviousValue: dbOrder.OrderNo,
		UpdatedAt:     time.Now().In(s.location),
		Message:       fmt.Sprintf("Schedule order completed with order number: %s", dbOrder.ShopifyDraftOrderID),
		Type:          "schedule_order",
	}

	// 3. update order in db
	dbOrder.ShopifyOrderID = strings.TrimPrefix(resp.DraftOrder.Order.ID, fmt.Sprintf("%s/%s/", shopify.ShopifyIDPrefix, shopify.OrderKind))
	// dbOrder.ID = resp.DraftOrder.Order.ID
	dbOrder.OrderNo = resp.DraftOrder.Order.Name
	dbOrder.ShopifyFinancialStatus = strings.ToLower(resp.DraftOrder.Order.DisplayFinancialStatus)
	dbOrder.ShopifyDraftOrderID = ""
	dbOrder.ShopifyDraftOrderLink = ""
	dbOrder.OrderFinalizedDate = &currentDate
	if err := s.DB.WithContext(ctx).Save(&dbOrder).Error; err != nil {
		s.Logger.Error("Error updating order", zap.Error(err))
		return
	}

	tx := s.DB.Begin()
	var errLog error
	defer db.DBRollback(tx, &errLog)

	errLog = helpers.AddLog(
		ctx,
		tx,
		log,
		dbOrder.ID,
	)
	if errLog != nil {
		s.Logger.Error(errLog.Error(), zap.Any("log", log))
	}

	if dbOrder.ShopifyFinancialStatus == "PAID" {
		jobs.JobQueue <- jobs.Job{
			OrderRequest: models.CreateOrderRequest{
				OrderID: dbOrder.OrderNo,
				Method:  "shopify_payments",
			},
		}
	}
}

// ProcessPaidOrderWebhook processes the paid order webhook from shopify
func (s *webhookService) ProcessPaidOrderWebhook(ctx context.Context, webhook models.OrderWebhook) {
	s.Logger.Info("Processing paid order webhook", zap.String("order_id", webhook.Name))

	jobs.JobQueue <- jobs.Job{
		OrderRequest: models.CreateOrderRequest{
			OrderID: webhook.Name,
			Method:  "shopify_payments",
		},
	}
}
