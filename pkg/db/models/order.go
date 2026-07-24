package models

import (
	"encoding/json"
	"time"
)

type Order struct {
	ID               string          `json:"id" gorm:"column:id"`
	OrderNo          string          `json:"orderNo" gorm:"column:order_no"`
	CustomerName     string          `json:"customerName" gorm:"column:customer_name"`
	CustomerEmail    string          `json:"customerEmail" gorm:"column:customer_email"`
	NextDeliveryDate string          `json:"nextDeliveryDate" gorm:"column:next_delivery_date"`
	LineItems        json.RawMessage `json:"lineItems" gorm:"column:lineitems"`
	ReceiptLink      string          `json:"receiptLink" gorm:"column:receipt_link"`
	Frequency        string          `json:"frequency" gorm:"column:frequency"`
	Address          json.RawMessage `json:"address" gorm:"column:address"`
	Status           string          `json:"status" gorm:"column:status"`
	LinkToMap        string          `json:"linkToMap" gorm:"column:link_to_map"`
	ShippingMethod   string          `json:"shippingMethod" gorm:"column:shipping_method"`
	Method           string          `json:"method" gorm:"column:method"`
	Coupon           string          `json:"coupon" gorm:"column:coupon"`
	Logs             json.RawMessage `json:"logs" gorm:"column:logs"`
	CreatedAt        time.Time       `json:"createdAt" gorm:"column:created_at"`
	ActivationDate   *time.Time      `json:"activationDate" gorm:"column:activation_date"` // date draft created
	// DeliveryCompletedDate    *time.Time      `json:"deliveryCompletedDate" gorm:"column:delivery_completed_date"`
	OrderFinalizedDate *time.Time `json:"orderFinalizedDate" gorm:"column:order_finalized_date"` // date of draft order converted to regular order
	// PaymentCompletedDate     *time.Time      `json:"paymentCompletedDate" gorm:"column:payment_completed_date"`
	// ShopifyFulfillmentStatus string          `json:"shopifyFulfillmentID" gorm:"column:shopify_fulfillment_status"`
	ShopifyFinancialStatus string `json:"shopifyFinancialStatus" gorm:"column:shopify_financial_status"`
	ShopifyOrderID         string `json:"shopifyOrderID" gorm:"column:shopify_order_id"`
	ShopifyDraftOrderID    string `json:"shopifyDraftOrderID" gorm:"column:shopify_draft_order_id"`
	ShopifyDraftOrderLink  string `json:"shopifyDraftOrderLink" gorm:"column:shopify_draft_order_link"`
	IsDirectDebitAccount   bool   `json:"isDirectDebitAccount" gorm:"column:is_direct_debit_account"`
}

func (Order) TableName() string {
	return "orders"
}
