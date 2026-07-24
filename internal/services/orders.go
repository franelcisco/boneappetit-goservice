package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"bone_appetit_suscription/internal/domains"
	"bone_appetit_suscription/internal/models"
	helpers "bone_appetit_suscription/pkg"
	"bone_appetit_suscription/pkg/db"
	dbModels "bone_appetit_suscription/pkg/db/models"
	"bone_appetit_suscription/pkg/foodapi"
	"bone_appetit_suscription/pkg/mailgun"
	"bone_appetit_suscription/pkg/shopify"
)

const (
	onDemandURL               = "https://phpstack-1368391-5042796.cloudwaysapps.com/api/supabase/ondemand/order"
	emailTemplateConfirmation = "order_confirmation"
	emailTemplateReminder     = "order_reminder"
	emailTemplateDraft        = "draft_order_created"
	emailTemplateNotUpdated   = "subscription_not_updated"

	directDebitAccountOrderFirtsTag     = "direct_debit_account_firts"
	directDebitAccountOrderRecurrentTag = "direct_debit_account_recurrent"
)

var emailsTemplates = map[string]mailgun.SendEmailRequest{
	"order_confirmation": {
		Subject:  "Tu próxima orden ha sido programada exitosamente 💛",
		Template: "next order scheduled success",
	},
	"order_reminder": {
		Subject:  "Mañana se generará tu próxima orden 💛",
		Template: "one day before reminder email",
	},
	"draft_order_created": {
		Subject:  "Completa tu orden para enviarte 💛",
		Template: "draft order created",
	},
	"subscription_not_updated": {
		Subject:  "Aún no tenemos una orden programada asociada a este correo 🥺",
		Template: "subscription not updated",
	},
}

// testUserEmails = []string{"celmira@boneappetit.food", "francisco@boneappetit.food", "cesardejesuspelusa@gmail.com", "rcjfrancisco@gmail.com"}

type orderService struct {
	shopifyRepository shopify.Repository
	logger            *zap.Logger
	db                *gorm.DB
	location          *time.Location
	muRepository      mailgun.Repository
	foodAPI           *foodapi.Client
}

func NewOrderService(
	db *gorm.DB,
	shopifyRepository shopify.Repository,
	muRepository mailgun.Repository,
	foodAPI *foodapi.Client,
	location *time.Location,
	logger *zap.Logger,
) domains.OrderService {
	return &orderService{
		shopifyRepository: shopifyRepository,
		logger:            logger,
		db:                db,
		muRepository:      muRepository,
		foodAPI:           foodAPI,
		location:          location,
	}
}

// sendEmail sends an email to the customer
func (o *orderService) sendEmail(
	ctx context.Context,
	vars models.ConfirmationOrderEmailVars,
	email string,
	template string,
) error {
	varsForEmail := helpers.GetVarsForConfirmationOrderEmail(vars)

	emailTemplate := emailsTemplates[template]
	emailTemplate.Vars = varsForEmail
	emailTemplate.To = email

	err := o.muRepository.SendEmail(ctx, emailTemplate)
	if err != nil {
		// o.logger.Error(err.Error(), zap.String("to", order.Order.Customer.Email))
		return err
	}

	return nil
}

// createOrderInShopify creates an order in Shopify for the given user and line items
func (o *orderService) createOrderInShopify(
	ctx context.Context,
	dbOrder dbModels.Order,
	currentDate time.Time,
) (*shopify.OrderCreateResponse, error) {
	orderData, err := o.shopifyRepository.GetOrderByID(ctx, dbOrder.ID)
	if err != nil {
		return nil, err
	}

	var lineItems shopify.LineItemsEdge
	if err := json.Unmarshal(dbOrder.LineItems, &lineItems); err != nil {
		o.logger.Error(err.Error(), zap.Any("lineItems", dbOrder.LineItems))
		return nil, err
	}

	requestLineItems := make([]shopify.LineItemsNodeRequest, 0, len(lineItems.Edges))
	for _, item := range lineItems.Edges {
		if item.Node.Quantity > 0 && item.Node.Name != "Tip" && item.Node.Variant != nil {
			requestLineItems = append(requestLineItems, shopify.LineItemsNodeRequest{
				VariantID: item.Node.Variant.ID,
				Quantity:  item.Node.Quantity,
			})
		}
	}

	var tags []string
	re := regexp.MustCompile(`^(one_week|\d+_weeks)$`)
	for _, tag := range orderData.Order.Tags {
		if !re.MatchString(tag) {
			tags = append(tags, tag)
		}
	}
	tags = append(tags, dbOrder.Frequency)

	shopifyOrderRequest := shopify.CreateOrderInShopifyRequest{
		Order: shopify.CreateOrderInput{
			CustomerID:      orderData.Order.Customer.ID,
			Email:           dbOrder.CustomerEmail,
			Tags:            tags,
			LineItems:       requestLineItems,
			Note:            "Order created for manual subscription recurring payment",
			FinancialStatus: shopify.FinancialStatusPaid,
		},
	}

	order, err := o.shopifyRepository.CreateOrder(ctx, shopifyOrderRequest)
	if err != nil {
		o.logger.Error(err.Error(), zap.Any("request", shopifyOrderRequest))
		return nil, err
	}

	return order, nil
}

// createDraftOrderInShopify creates a draft order in shopify
func (o *orderService) createDraftOrderInShopify(
	ctx context.Context,
	dbOrder dbModels.Order,
	currentDate time.Time,
	jobs chan Job,
) (*shopify.DraftOrder, error) {
	// 1. Get order from shopify
	order, err := o.shopifyRepository.GetOrderByID(ctx, dbOrder.ID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		varsForEmail := models.ConfirmationOrderEmailVars{
			CustomerName: dbOrder.CustomerName,
		}
		err = o.sendEmail(
			ctx,
			varsForEmail,
			dbOrder.CustomerEmail,
			emailTemplateNotUpdated,
		)
		if err != nil {
			o.logger.Error(err.Error(), zap.String("orderId", dbOrder.ID))
		}
		o.logger.Error("order not found in shopify", zap.String("orderId", dbOrder.ID))
		return nil, errors.New("order not found in shopify")
	}

	var lineItems shopify.LineItemsEdge
	if err := json.Unmarshal(dbOrder.LineItems, &lineItems); err != nil {
		o.logger.Error(err.Error(), zap.Any("lineItems", dbOrder.LineItems))
		return nil, err
	}

	var requestLineItems []shopify.LineItemsNodeDraft
	for _, item := range lineItems.Edges {
		if item.Node.Quantity > 0 && item.Node.Name != "Tip" && item.Node.Variant != nil {
			requestLineItems = append(requestLineItems, shopify.LineItemsNodeDraft{
				VariantID: item.Node.Variant.ID,
				Quantity:  item.Node.Quantity,
			})
		}
	}

	if requestLineItems == nil {
		o.logger.Error("no valid line items found", zap.String("orderId", dbOrder.ID))
		return nil, errors.New("no valid line items found")
	}

	var tags []string
	re := regexp.MustCompile(`^(one_week|\d+_weeks)$`)
	for _, tag := range order.Order.Tags {
		if tag == directDebitAccountOrderFirtsTag {
			continue
		}

		if !re.MatchString(tag) {
			tags = append(tags, tag)
		}
	}
	tags = append(tags, dbOrder.Frequency)

	if dbOrder.IsDirectDebitAccount {
		tags = append(tags, directDebitAccountOrderRecurrentTag)
	}

	var address *shopify.CustomerAddress
	if err := json.Unmarshal(dbOrder.Address, &address); err != nil {
		o.logger.Error("error unmarshalling address", zap.Error(err), zap.Any("address", dbOrder.Address), zap.String("orderId", dbOrder.ID))
		return nil, err
	}

	var firstName, lastName string
	if customerNameList := strings.Split(dbOrder.CustomerName, " "); len(customerNameList) > 1 {
		firstName = customerNameList[0]
		lastName = customerNameList[1]
	} else {
		firstName = dbOrder.CustomerName
	}

	draftOrder := shopify.CreateDraftOrderInShopifyRequest{
		Input: shopify.CreateDraftOrderInput{
			AcceptAutomaticDiscounts: true,
			CustomerID:               order.Order.Customer.ID,
			Email:                    dbOrder.CustomerEmail,
			Tags:                     tags,
			TaxExempt:                false,
			ShippingLine: shopify.ShippingLineDraft{
				Title: dbOrder.ShippingMethod,
				Price: 1.99,
			},
			ShippingAddress: shopify.ShippingAddress{
				FirstName: firstName,
				LastName:  lastName,
				Phone:     address.Phone,
				Address1:  address.Address1,
				City:      address.City,
				Province:  address.Province,
				Country:   address.Country,
				Zip:       address.Zip,
			},
			BillingAddress: shopify.ShippingAddress{
				FirstName: firstName,
				LastName:  lastName,
				Phone:     address.Phone,
				Address1:  address.Address1,
				City:      address.City,
				Province:  address.Province,
				Country:   address.Country,
				Zip:       address.Zip,
			},
			LineItems: requestLineItems,
			CustomAttributes: []shopify.Metafield{
				{
					Key:   "Link to Map",
					Value: dbOrder.LinkToMap,
				},
			},
		},
	}

	if dbOrder.Coupon != "" {
		draftOrder.Input.DiscountCodes = strings.Split(dbOrder.Coupon, ",")
	}
	// 2. Create draft order in shopify
	resp, err := o.shopifyRepository.CreateDraftOrder(ctx, draftOrder)
	if err != nil {
		o.logger.Error(err.Error(), zap.String("orderId", dbOrder.ID))
		return nil, err
	}

	// 3. Update dbOrder with Shopify draft order details
	if err := o.updateOrder(
		ctx, dbOrder, currentDate, resp.Name, resp.InvoiceURL,
	); err != nil {
		return nil, err
	}

	if err := o.sendDraftEmail(
		ctx, address, dbOrder, lineItems.Edges, resp.InvoiceURL,
	); err != nil {
		return nil, err
	}

	if dbOrder.IsDirectDebitAccount && jobs != nil {
		jobs <- Job{dbOrder: dbOrder, draftOrder: *resp}
	}

	return resp, nil
}

// updateOrder updates the order in the database and shopify if necessary
func (o *orderService) updateOrder(
	ctx context.Context,
	dbOrder dbModels.Order,
	currentDate time.Time,
	orderName,
	orderURL string,
) error {
	dbOrder.Status = "Triggered"
	dbOrder.ShopifyDraftOrderID = orderName
	dbOrder.ShopifyDraftOrderLink = orderURL
	dbOrder.ActivationDate = &currentDate
	dbOrder.OrderFinalizedDate = nil
	dbOrder.ShopifyFinancialStatus = "pending"
	err := o.db.WithContext(ctx).Save(&dbOrder).Error
	if err != nil {
		o.logger.Error(err.Error(), zap.String("orderId", dbOrder.ID))
		return err
	}

	return nil
}

// sendDraftEmail sends an email to the customer with the draft order details
func (o *orderService) sendDraftEmail(
	ctx context.Context,
	address *shopify.CustomerAddress,
	dbOrder dbModels.Order,
	lineItems []shopify.LineItemsNode,
	orderURL string,
) error {
	var customerAddress *models.CustomerAddress
	if address != nil {
		customerAddress = &models.CustomerAddress{
			Address1: address.Address1,
			Address2: address.Address2,
			City:     address.City,
			Province: address.Province,
			Country:  address.Country,
			Zip:      address.Zip,
		}
	}

	items := helpers.GetLineItemsForEmail(lineItems)
	varsForEmail := models.ConfirmationOrderEmailVars{
		OrderURL:         orderURL,
		IsDraft:          true,
		OrderNo:          dbOrder.OrderNo,
		NextDeliveryDate: dbOrder.NextDeliveryDate,
		Method:           dbOrder.ShippingMethod,
		Frequency:        dbOrder.Frequency,
		Items:            items,
		CustomerName:     dbOrder.CustomerName,
		CustomerAddress:  customerAddress,
	}

	// 4. Send email
	err := o.sendEmail(
		ctx,
		varsForEmail,
		dbOrder.CustomerEmail,
		emailTemplateDraft,
	)
	if err != nil {
		o.logger.Error(err.Error(), zap.String("orderId", dbOrder.ID))
		return err
	}

	return nil
}

// createOrderInDB creates an order record in the database
func (o *orderService) upsertOrderInDB(
	ctx context.Context,
	tx *gorm.DB,
	order *shopify.Order,
	nextDeliveryDate string,
	frequencyTag string,
	receiptLink string,
	method string,
) error {
	orderLines, err := json.Marshal(order.LineItems)
	if err != nil {
		o.logger.Error(err.Error(), zap.Any("lineItems", order.LineItems))
		return err
	}

	address, err := json.Marshal(order.Customer.DefaultAddress)
	if err != nil {
		o.logger.Error(err.Error(), zap.Any("address", order.Customer.DefaultAddress))
		return err
	}

	var linkToMap string
	for _, attr := range order.CustomAttributes {
		if attr.Key == "Link to Map" {
			linkToMap = attr.Value
			break
		}
	}

	var isDirectDebitAccount bool
	if slices.Contains(order.Tags, directDebitAccountOrderFirtsTag) {
		isDirectDebitAccount = true
	} else {
		var existing dbModels.Order
		if err := tx.WithContext(ctx).
			Model(&dbModels.Order{}).
			Select("is_direct_debit_account").
			Where("order_no = ?", order.Name).
			First(&existing).Error; err == nil && existing.IsDirectDebitAccount {
			isDirectDebitAccount = true
		}
	}

	currentDate := time.Now().In(o.location)
	err = tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "order_no"}},
		UpdateAll: true,
	}).Create(&dbModels.Order{
		Status:           "Active",
		ID:               order.ID,
		OrderNo:          order.Name,
		CustomerName:     order.Customer.DisplayName,
		CustomerEmail:    order.Customer.Email,
		NextDeliveryDate: nextDeliveryDate,
		LineItems:        orderLines,
		Frequency:        frequencyTag,
		ReceiptLink:      receiptLink,
		Address:          address,
		LinkToMap:        linkToMap,
		Method:           method,
		ShippingMethod:   order.ShippingLine.Title,
		CreatedAt:        currentDate,
		Logs:             nil,
		// PaymentCompletedDate:     &currentDate,
		// ShopifyFulfillmentStatus: order.DisplayFulfillmentStatus,
		ShopifyFinancialStatus: strings.ToLower(order.DisplayFinancialStatus),
		ShopifyOrderID:         strings.TrimPrefix(order.ID, fmt.Sprintf("%s/%s", shopify.ShopifyIDPrefix, shopify.OrderKind)),
		IsDirectDebitAccount:   isDirectDebitAccount,
	}).Error
	if err != nil {
		o.logger.Error(err.Error())
		return err
	}

	return nil
}

// processDirectDebitAccountOrder delegates the R4 charge to food-api-dev and
// orchestrates the post-charge actions (complete draft, clear local flag).
func (o *orderService) processDirectDebitAccountOrder(
	ctx context.Context,
	dbOrder dbModels.Order,
	draftOrderID string,
) error {
	draftNum := strings.TrimPrefix(draftOrderID, shopify.GID(shopify.DraftOrderKind, ""))

	resp, err := o.foodAPI.DirectDebitAccountRecurring(ctx, foodapi.RecurringRequest{
		DraftOrderID: draftNum,
	})
	if err != nil {
		o.logger.Error("food-api recurring charge call failed",
			zap.String("orderId", dbOrder.ID),
			zap.String("draftOrderId", draftOrderID),
			zap.Error(err))
		return err
	}

	if !resp.Affiliate {
		if err := o.db.WithContext(ctx).Model(&dbModels.Order{}).
			Where("customer_email = ?", dbOrder.CustomerEmail).
			Update("is_direct_debit_account", false).Error; err != nil {
			o.logger.Error("failed to clear local is_direct_debit_account flag",
				zap.String("customerEmail", dbOrder.CustomerEmail),
				zap.Error(err))
		}
	}

	if resp.Success {
		if err := o.completeDraftOrder(ctx, draftOrderID); err != nil {
			o.logger.Error("error completing draft order",
				zap.String("orderId", dbOrder.ID),
				zap.String("draftOrderId", draftOrderID),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// completeDraftOrder completes the draft order in shopify
func (o *orderService) completeDraftOrder(
	ctx context.Context,
	draftOrderID string,
) error {
	err := o.shopifyRepository.CompleteDraftOrder(ctx, draftOrderID)
	if err != nil {
		return err
	}

	err = o.markDraftOrderAsPaid(ctx, draftOrderID)
	if err != nil {
		return err
	}

	return nil
}

// markDraftOrderAsPaid marks the draft order as paid in shopify
func (o *orderService) markDraftOrderAsPaid(
	ctx context.Context,
	draftOrderID string,
) error {
	order, err := o.shopifyRepository.GetDraftOrderByID(ctx, draftOrderID)
	if err != nil {
		return err
	}

	if order.DraftOrder.Order == nil {
		o.logger.Error("draft order does not have an associated order", zap.String("draftOrderId", draftOrderID))
		return fmt.Errorf("draft order does not have an associated order")
	}

	err = o.shopifyRepository.MarkOrderAsPaid(ctx, order.DraftOrder.Order.ID)
	if err != nil {
		return err
	}

	return nil
}

const workers = 50

type Job struct {
	dbOrder    dbModels.Order
	draftOrder shopify.DraftOrder
}

const directDebitAccountDefaultTimeout = 130 * time.Second

func (o *orderService) startPool(poolSize int, jobs <-chan Job) {
	sem := make(chan struct{}, poolSize) // Limit number of workers
	workerID := 1
	for job := range jobs {
		sem <- struct{}{}                                                                    // Acquire lifeguard
		ctx, cancel := context.WithTimeout(context.TODO(), directDebitAccountDefaultTimeout) // Set timeout for each job
		workerID++
		go func(ctx context.Context, cancel context.CancelFunc, job Job, workerID int) {
			defer func() { <-sem }() // Release lifeguard
			defer cancel()
			o.logger.Info("starting job for order", zap.Int("worker", workerID), zap.String("orderId", job.dbOrder.ID))
			err := o.processDirectDebitAccountOrder(ctx, job.dbOrder, job.draftOrder.ID)
			if err != nil {
				o.logger.Error("error processing direct debit account order", zap.String("orderId", job.dbOrder.ID), zap.Error(err))
			}

			o.logger.Info("finished job for order", zap.Int("worker", workerID), zap.String("orderId", job.dbOrder.ID))
		}(ctx, cancel, job, workerID)
	}
}

// ScheduleNextOrder schedules the next order for a subscription
func (o *orderService) ScheduleNextOrder(
	ctx context.Context,
	req models.CreateOrderRequest,
) error {
	orders, err := o.shopifyRepository.GetOrderByQuery(ctx, shopify.QueryOrderFilter{
		Name: req.OrderID,
	}, 1)
	if err != nil {
		return err
	}

	if orders == nil || len(orders.Orders.Nodes) == 0 {
		o.logger.Error("order not found in shopify", zap.String("orderId", req.OrderID))
		return errors.New("order not found in shopify")
	}

	order := &orders.Orders.Nodes[0]

	if helpers.IsAutomaticSubscription(order.Tags) {
		o.logger.Info("is automatic subscription", zap.Any("tags", order.Tags), zap.String("order", order.Name))
		return nil
	}

	if len(order.Tags) == 0 && helpers.IsShopifyPaymentGateway(order.PaymentGatewayNames) {
		o.logger.Error("no tags in shopify payment", zap.Any("paymentGateway", order.PaymentGatewayNames), zap.Any("tags", order.Tags), zap.String("order", order.Name))
	}

	// createAt is 2025-10-14T20:26:42Z
	orderCreatedDate, err := time.Parse(time.RFC3339, order.CreatedAt)
	if err != nil {
		o.logger.Error(err.Error(), zap.String("createdAt", order.CreatedAt), zap.Error(err))
		return err
	}

	var frequencyTag string
	re := regexp.MustCompile(`^(one_week|\d+_weeks)$`)
	for _, tag := range order.Tags {
		if re.MatchString(tag) {
			frequencyTag = tag
			break
		}
	}

	if frequencyTag == "" {
		frequencyTag = "2_weeks"
	}

	nextDeliveryDate, err := helpers.CalculateNextDeliveryDate(orderCreatedDate, frequencyTag)
	if err != nil {
		o.logger.Error(err.Error(), zap.String("orderId", req.OrderID), zap.Error(err))
		return errors.New("failed to calculate next delivery date")
	}

	var (
		tx    = o.db.Begin()
		errDB error
	)
	defer db.DBRollback(tx, &errDB)

	// 3. createOrderInDB
	errDB = o.upsertOrderInDB(
		ctx,
		tx,
		order,
		nextDeliveryDate.Format("02/01/2006"),
		frequencyTag,
		req.Receipt,
		req.Method,
	)
	if errDB != nil {
		return errDB
	}

	items := helpers.GetLineItemsForEmail(order.LineItems.Edges)
	varsForEmail := models.ConfirmationOrderEmailVars{
		OrderURL:         fmt.Sprintf("%s/%s", onDemandURL, strings.TrimPrefix(order.Name, "#")),
		IsDraft:          false,
		OrderNo:          order.Name,
		NextDeliveryDate: nextDeliveryDate.Format("02/01/2006"),
		Method:           req.Method,
		Frequency:        frequencyTag,
		Items:            items,
		CustomerName:     order.Customer.DisplayName,
		CustomerAddress: &models.CustomerAddress{
			Address1: order.Customer.DefaultAddress.Address1,
			Address2: order.Customer.DefaultAddress.Address2,
			City:     order.Customer.DefaultAddress.City,
			Province: order.Customer.DefaultAddress.Province,
			Country:  order.Customer.DefaultAddress.Country,
			Zip:      order.Customer.DefaultAddress.Zip,
		},
	}

	// 5. SendConfirmationEmail
	err = o.sendEmail(
		ctx,
		varsForEmail,
		order.Customer.Email,
		emailTemplateConfirmation,
	)
	if err != nil {
		o.logger.Error("send email", zap.Error(err), zap.Any("template", emailTemplateConfirmation), zap.Any("vars", varsForEmail), zap.Any("email", order.Customer.Email))
	}

	return nil
}

// CreateOnDemandOrderProcess creates an on-demand order in shopify and the database
func (o *orderService) CreateOnDemandOrderProcess(
	ctx context.Context,
	orderNo string,
) (string, error) {
	var (
		currentDate = time.Now().In(o.location)
		order       dbModels.Order
	)
	// 1. Get order from DB
	if err := o.db.WithContext(ctx).Model(&dbModels.Order{}).
		Where("order_no = ?", fmt.Sprintf("#%s", orderNo)).
		Where("status = ?", "Active").
		First(&order).Error; err != nil {
		o.logger.Error(err.Error(), zap.String("orderNo", orderNo))
		return "", err
	}

	// 2. Create order in shopify
	resp, err := o.createDraftOrderInShopify(ctx, order, currentDate, nil)
	if err != nil {
		return "", err
	}

	if resp == nil {
		o.logger.Error("failed to create order in shopify", zap.String("orderId", order.ID))
		return "", fmt.Errorf("failed to create order in shopify")
	}

	tx := o.db.Begin()
	var errLog error
	defer db.DBRollback(tx, &errLog)

	errLog = helpers.AddLog(
		ctx,
		tx,
		models.Log{
			UpdatedField:  "on_demand_order_created",
			NewValue:      resp.Name,
			PreviousValue: order.ShopifyDraftOrderID,
			UpdatedAt:     time.Now().In(o.location),
			Message:       fmt.Sprintf("Schedule order created with order number: %s", resp.Name),
			Type:          "schedule_order",
		},
		order.ID,
	)
	if errLog != nil {
		o.logger.Error(errLog.Error(), zap.String("OrderNo", order.OrderNo))
	}

	o.logger.Info("draft order created successfully", zap.String("draftOrderId", resp.ID), zap.String("OrderNo", order.OrderNo))

	return resp.InvoiceURL, nil
}

// scheduleOrders schedules orders for the current date
func (o *orderService) ScheduleOrders(
	ctx context.Context,
) error {
	o.logger.Info("scheduling orders...")
	var (
		currentDate = time.Now().In(o.location)
		dbOrders    []dbModels.Order
	)

	if err := o.db.WithContext(ctx).Model(&dbModels.Order{}).
		Where("status = ?", "Active").
		Where("next_delivery_date = ? OR next_delivery_date = ?",
			currentDate.Format("02/01/2006"),
			currentDate.Format("2/01/2006"),
		).
		Find(&dbOrders).Error; err != nil {
		o.logger.Error(err.Error())
		return err
	}

	jobs := make(chan Job)
	go o.startPool(workers, jobs)
	for _, dbOrder := range dbOrders {
		resp, err := o.createDraftOrderInShopify(ctx, dbOrder, currentDate, jobs)
		if err != nil {
			continue
		}

		tx := o.db.Begin()
		errLog := helpers.AddLog(
			ctx,
			tx,
			models.Log{
				UpdatedField:  "schedule_order_created",
				NewValue:      resp.Name,
				PreviousValue: dbOrder.ShopifyDraftOrderID,
				UpdatedAt:     time.Now().In(o.location),
				Message:       fmt.Sprintf("Schedule order created with order number: %s", resp.Name),
				Type:          "schedule_order",
			},
			dbOrder.ID,
		)
		if errLog != nil {
			o.logger.Error(errLog.Error(), zap.Any("id", dbOrder.ID))
		}
		db.DBRollback(tx, &errLog)
	}
	close(jobs)

	o.logger.Info("orders scheduled successfully", zap.Int("count", len(dbOrders)))
	return nil
}

// ReminderProcess sends reminder emails for orders scheduled for tomorrow
func (o *orderService) ReminderProcess(
	ctx context.Context,
) error {
	o.logger.Info("Sending reminder emails...")
	tomorrowDate := time.Now().In(o.location).AddDate(0, 0, 1).Format("02/01/2006")
	var orders []dbModels.Order
	if err := o.db.WithContext(ctx).Model(&dbModels.Order{}).
		Where("next_delivery_date = ? AND status = ?", tomorrowDate, "Active").
		Find(&orders).Error; err != nil {
		o.logger.Error(err.Error())
		return err
	}

	for _, order := range orders {
		var lineItems shopify.LineItemsEdge
		if err := json.Unmarshal(order.LineItems, &lineItems); err != nil {
			o.logger.Error(err.Error(), zap.Any("lineItems", order.LineItems))
			continue
		}

		var customerAddress shopify.CustomerAddress
		if err := json.Unmarshal(order.Address, &customerAddress); err != nil {
			o.logger.Error(err.Error(), zap.Any("address", order.Address))
			continue
		}

		items := helpers.GetLineItemsForEmail(lineItems.Edges)

		varsForEmail := models.ConfirmationOrderEmailVars{
			OrderURL:         fmt.Sprintf("%s/%s", onDemandURL, strings.TrimPrefix(order.OrderNo, "#")),
			IsDraft:          false,
			OrderNo:          order.OrderNo,
			NextDeliveryDate: order.NextDeliveryDate,
			Method:           order.ShippingMethod,
			Frequency:        order.Frequency,
			Items:            items,
			CustomerName:     order.CustomerName,
			CustomerAddress: &models.CustomerAddress{
				Address1: customerAddress.Address1,
				Address2: customerAddress.Address2,
				City:     customerAddress.City,
				Province: customerAddress.Province,
				Country:  customerAddress.Country,
				Zip:      customerAddress.Zip,
			},
		}

		if err := o.sendEmail(
			ctx,
			varsForEmail,
			order.CustomerEmail,
			emailTemplateReminder,
		); err != nil {
			o.logger.Error(err.Error(), zap.String("orderId", order.ID))
			continue
		}
	}

	o.logger.Info("reminder emails sent successfully", zap.Int("count", len(orders)))
	return nil
}
