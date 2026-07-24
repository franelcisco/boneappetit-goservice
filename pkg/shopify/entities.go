package shopify

import (
	"encoding/json"
)

type gqlRequest struct {
	Query     string `json:"query"`
	Variables any    `json:"variables"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// emun shopify kind
const (
	OrderKind       = "Order"
	CustomerKind    = "Customer"
	DraftOrderKind  = "DraftOrder"
	ShopifyIDPrefix = "gid://shopify"
)

// GetOrderByIDResponse constructs a global ID for Shopify entities
type GetOrderByIDResponse struct {
	Order *Order `json:"order"`
}

// GetOrderByQueryResponse represents the response for querying multiple orders
type GetOrderByQueryResponse struct {
	Orders OrdersNodes `json:"orders"`
}

// OrdersNodes represents a list of order nodes
type OrdersNodes struct {
	Nodes []Order `json:"nodes"`
}

// OrdersByPage represents a paginated list of orders
type OrdersByPage struct {
	Order
	PageInfo PageInfo `json:"pageInfo"`
}

// PageInfo represents pagination information for a list of orders
type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	StartCursor     string `json:"startCursor"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	EndCursor       string `json:"endCursor"`
}

// Order represents a Shopify order
type Order struct {
	ID                       string            `json:"id"`
	Name                     string            `json:"name"`
	Tags                     []string          `json:"tags"`
	CreatedAt                string            `json:"createdAt"`
	DisplayFinancialStatus   string            `json:"displayFinancialStatus"`
	DisplayFulfillmentStatus string            `json:"displayFulfillmentStatus"`
	TotalPriceSet            ShopMoney         `json:"totalPriceSet"`
	LineItems                LineItemsEdge     `json:"lineItems"`
	Customer                 Customer          `json:"customer"`
	CustomAttributes         []CustomAttribute `json:"customAttributes"`
	ShippingLine             ShippingLine      `json:"shippingLine"`
	PaymentGatewayNames      []string          `json:"paymentGatewayNames"`
}

// CustomAttribute represents a custom attribute in an order
type CustomAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ShippingLine represents the shipping line of an order
type ShippingLine struct {
	Title     string `json:"title"`
	ID        string `json:"id"`
	IsRemoved bool   `json:"isRemoved"`
}

// LineItemsEdge represents the edge of line items in an order
type LineItemsEdge struct {
	Edges []LineItemsNode `json:"edges"`
}

// LineItemsNode represents a node in the line items edge
type LineItemsNode struct {
	Node LineItem `json:"node"`
}

// LineItem represents a line item in an order
type LineItem struct {
	Title    string   `json:"title"`
	Name     string   `json:"name"`
	Quantity int      `json:"quantity"`
	SKU      string   `json:"sku"`
	Variant  *Variant `json:"variant"`
}

// Variant represents a product variant
type Variant struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ShopMoney represents the total or subtotal price set of an order
type ShopMoney struct {
	ShopMoney ShopMoneyProps `json:"shopMoney"`
}

// ShopMoneyProps represents an amount of money in a specific currency
type ShopMoneyProps struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}

// Customer represents a Shopify customer
type Customer struct {
	ID                 string                      `json:"id"`
	DisplayName        string                      `json:"displayName"`
	Email              string                      `json:"email"`
	DefaultPhoneNumber *CustomerDefaultPhoneNumber `json:"defaultPhoneNumber"`
	ParentID           *Metafield                  `json:"parentId"`
	DefaultAddress     *CustomerAddress            `json:"defaultAddress"`
	DirectDebitAccount *Metafield                  `json:"directDebitAccount"`
}

type CustomerAddress struct {
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	Province string `json:"province"`
	Country  string `json:"country"`
	Zip      string `json:"zip"`
	Phone    string `json:"phone"`
}

type CustomerDefaultPhoneNumber struct {
	PhoneNumber string `json:"phoneNumber"`
}

// Metafield represents a Shopify metafield
type Metafield struct {
	Key       string          `json:"key"`
	Value     string          `json:"value"`
	JsonValue json.RawMessage `json:"jsonValue,omitempty"`
}

type DirectDebitAccount struct {
	Account string `json:"account"`
	DNI     string `json:"dni"`
}

// QueryOrderFilter represents the filters for querying orders
type QueryOrderFilter struct {
	Name string
}

// SetCustomerMetafieldResponse
type SetCustomerMetafieldResponse struct {
	CustomerUpdate
}

type CustomerUpdate struct {
	Customer   Customer     `json:"customer"`
	UserErrors []UserErrors `json:"userErrors"`
}

type UserErrors struct {
	Message string `json:"message"`
}

// GetCustomerMetafieldResponse
type GetCustomerMetafieldResponse struct {
	Customer struct {
		Metafield *Metafield `json:"metafield"`
	} `json:"customer"`
}

type MarkOrderAsPaidResponse struct {
	Order      Order       `json:"order"`
	UserErrors *UserErrors `json:"userErrors"`
}

type CreateDraftOrderResponse struct {
	DraftOrderCreate struct {
		DraftOrder DraftOrder `json:"draftOrder"`
	} `json:"draftOrderCreate"`
	UserErrors []UserErrors `json:"userErrors"`
}

type DraftOrder struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	InvoiceURL    string    `json:"invoiceUrl"`
	Order         *Order    `json:"order,omitempty"`
	Customer      Customer  `json:"customer"`
	TotalPriceSet ShopMoney `json:"totalPriceSet"`
}

type CreateDraftOrderInShopifyRequest struct {
	Input CreateDraftOrderInput `json:"input"`
}

type CreateDraftOrderInput struct {
	AcceptAutomaticDiscounts bool                 `json:"acceptAutomaticDiscounts"`
	CustomerID               string               `json:"customerId"`
	Email                    string               `json:"email"`
	Tags                     []string             `json:"tags"`
	TaxExempt                bool                 `json:"taxExempt"`
	ShippingLine             ShippingLineDraft    `json:"shippingLine"`
	ShippingAddress          ShippingAddress      `json:"shippingAddress"`
	BillingAddress           ShippingAddress      `json:"billingAddress"`
	LineItems                []LineItemsNodeDraft `json:"lineItems"`
	CustomAttributes         []Metafield          `json:"customAttributes"`
	DiscountCodes            []string             `json:"discountCodes,omitempty"`
}

type ShippingAddress struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Address1  string `json:"address1"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Country   string `json:"country"`
	Zip       string `json:"zip"`
}

// ShippingLine represents the shipping line of an order
type ShippingLineDraft struct {
	Title string  `json:"title"`
	Price float64 `json:"price"`
}

type LineItemsNodeDraft struct {
	VariantID string `json:"variantId"`
	Quantity  int    `json:"quantity"`
}

type DraftOrderData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GetDraftOrderByIDResponse struct {
	DraftOrder *DraftOrder `json:"draftOrder"`
}

type OrderCreateResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	StatusPageURL string    `json:"statusPageUrl"`
	TotalPriceSet ShopMoney `json:"totalPriceSet"`
}

type CreateOrderResponse struct {
	OrderCreate struct {
		Order OrderCreateResponse `json:"order"`
	} `json:"orderCreate"`
	UserErrors []UserErrors `json:"userErrors"`
}

type FinancialStatusType string

const (
	FinancialStatusPaid    FinancialStatusType = "PAID"
	FinancialStatusPending FinancialStatusType = "PENDING"
)

type CreateOrderInput struct {
	CustomerID      string                 `json:"customerId"`
	Email           string                 `json:"email"`
	Tags            []string               `json:"tags"`
	LineItems       []LineItemsNodeRequest `json:"lineItems"`
	Note            string                 `json:"note,omitempty"`
	FinancialStatus FinancialStatusType    `json:"financialStatus,omitempty"`
}

type LineItemsNodeRequest struct {
	VariantID string `json:"variantId"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderInShopifyRequest struct {
	Order CreateOrderInput `json:"order"`
}

type CompleteDraftOrderResponse struct {
	UserErrors []UserErrors `json:"userErrors"`
}

