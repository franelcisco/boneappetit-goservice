package shopify

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bone_appetit_suscription/internal/domains"

	"go.uber.org/zap"
)

const (
	customerNamespace             = "customer_fields"
	customNamespace               = "custom"
	customerParentKey             = "parent_id"
	customerDirectDebitKey        = "direct_debit"
	customerDirectDebitAccountKey = "direct_debit_account"
)

// Repository defines methods to interact with Shopify API
type Repository interface {
	GetOrderByID(ctx context.Context, id string) (*GetOrderByIDResponse, error)
	GetOrderByQuery(ctx context.Context, filters QueryOrderFilter, first int) (*GetOrderByQueryResponse, error)
	SetCustomerParentID(ctx context.Context, customerID, parentID string) error
	GetCustomerParentID(ctx context.Context, customerID string) (*Metafield, error)
	GetCustomerDebitDirect(ctx context.Context, customerID string) (*Metafield, error)
	MarkOrderAsPaid(ctx context.Context, orderID string) error
	CreateDraftOrder(ctx context.Context, input any) (*DraftOrder, error)
	CreateOrder(ctx context.Context, input any) (*OrderCreateResponse, error)
	GetDraftOrderByID(ctx context.Context, id string) (*GetDraftOrderByIDResponse, error)
	CompleteDraftOrder(ctx context.Context, draftOrderID string) error
}

// Repository is a Shopify API repository
type repository struct {
	gql    *GraphQLClient
	Logger *zap.Logger
}

// NewRepository creates a new Shopify API repository
func NewRepository(
	shopDomain, apiVersion, adminToken string, logger *zap.Logger,
) Repository {
	return &repository{
		gql:    NewGraphQLClient(shopDomain, apiVersion, adminToken, logger),
		Logger: logger,
	}
}

func getQueryOrderByFilters(filters QueryOrderFilter) string {
	var query string

	if filters.Name != "" {
		if !strings.HasPrefix(filters.Name, "#") {
			filters.Name = fmt.Sprintf("#%s", filters.Name)
		}

		query += fmt.Sprintf("name:%s ", filters.Name)
	}

	return query
}

// GetOrderByID retrieves an order by its ID
func (r *repository) GetOrderByID(
	ctx context.Context, gid string,
) (*GetOrderByIDResponse, error) {
	if !strings.Contains(gid, OrderKind) {
		gid = GID(OrderKind, gid)
	}

	var resp GetOrderByIDResponse
	if err := r.gql.Do(ctx, getOrderByIDQuery, map[string]any{"id": gid}, &resp); err != nil {
		return nil, err
	}

	if resp.Order == nil {
		r.Logger.Error("order not found", zap.String("id", gid))
		return nil, fmt.Errorf("order %s not found", gid)
	}

	return &resp, nil
}

// GetOrderByQuery retrieves orders based on the provided filters
func (r *repository) GetOrderByQuery(
	ctx context.Context, filters QueryOrderFilter, first int,
) (*GetOrderByQueryResponse, error) {
	query := getQueryOrderByFilters(filters)
	var resp GetOrderByQueryResponse
	if err := r.gql.Do(ctx, getOrderByName, map[string]any{"query": query, "first": first}, &resp); err != nil {
		return nil, err
	}

	if len(resp.Orders.Nodes) == 0 {
		r.Logger.Error("no orders found for query", zap.String("query", query))
		return nil, domains.OrderNotFound
	}

	return &resp, nil
}

func (r *repository) SetCustomerParentID(
	ctx context.Context, customerID, parentID string,
) error {
	gid := GID(CustomerKind, customerID)
	vars := map[string]any{
		"id":        gid,
		"namespace": customerNamespace,
		"key":       customerParentKey,
		"value":     parentID,
	}
	var resp SetCustomerMetafieldResponse
	if err := r.gql.Do(ctx, setCustomerMetafield, vars, &resp); err != nil {
		return err
	}

	if len(resp.UserErrors) > 0 {
		r.Logger.Error("failed to set customer parent ID", zap.Any("errors", resp.UserErrors))
		return errors.New("failed to set customer parent ID")
	}

	return nil
}

func (r *repository) GetCustomerParentID(
	ctx context.Context, customerID string,
) (*Metafield, error) {
	gid := GID(CustomerKind, customerID)
	vars := map[string]any{
		"id":        gid,
		"namespace": customerNamespace,
		"key":       customerParentKey,
	}

	var resp GetCustomerMetafieldResponse
	if err := r.gql.Do(ctx, getCustomerMetafield, vars, &resp); err != nil {
		return nil, err
	}

	if resp.Customer.Metafield == nil {
		return nil, errors.New("customer parent ID metafield not found")
	}

	return resp.Customer.Metafield, nil
}

func (r *repository) GetCustomerDebitDirect(
	ctx context.Context, customerID string,
) (*Metafield, error) {
	gid := GID(CustomerKind, customerID)
	vars := map[string]any{
		"id":        gid,
		"namespace": customerNamespace,
		"key":       customerDirectDebitKey,
	}

	var resp GetCustomerMetafieldResponse
	if err := r.gql.Do(ctx, getCustomerMetafield, vars, &resp); err != nil {
		return nil, err
	}

	if resp.Customer.Metafield == nil {
		return nil, errors.New("customer direct debit metafield not found")
	}

	return resp.Customer.Metafield, nil
}

// MarkOrderAsPaid marks an order as paid
func (r *repository) MarkOrderAsPaid(ctx context.Context, gid string) error {
	if !strings.Contains(gid, OrderKind) {
		gid = GID(OrderKind, gid)
	}

	vars := map[string]any{
		"id": gid,
	}
	var resp MarkOrderAsPaidResponse
	if err := r.gql.Do(ctx, markOrderAsPaid, vars, &resp); err != nil {
		r.Logger.Error(err.Error(), zap.Any("vars", vars))
		return err
	}

	if resp.UserErrors != nil {
		r.Logger.Error("failed to mark order as paid", zap.Any("errors", resp.UserErrors))
		return errors.New("failed to mark order as paid")
	}

	return nil
}

// CreateDraftOrder creates a draft order
func (r *repository) CreateDraftOrder(
	ctx context.Context, input any,
) (*DraftOrder, error) {
	var resp CreateDraftOrderResponse
	if err := r.gql.Do(ctx, createDraftOrder, input, &resp); err != nil {
		return nil, err
	}

	if resp.UserErrors != nil {
		r.Logger.Error("failed to create draft order", zap.Any("errors", resp.UserErrors))
		return nil, errors.New("failed to create draft order")
	}

	return &resp.DraftOrderCreate.DraftOrder, nil
}

// GetDraftOrderByID retrieves a draft order by its ID
func (r *repository) GetDraftOrderByID(
	ctx context.Context, gid string,
) (*GetDraftOrderByIDResponse, error) {
	if !strings.Contains(gid, DraftOrderKind) {
		gid = GID(DraftOrderKind, gid)
	}

	var resp GetDraftOrderByIDResponse
	if err := r.gql.Do(ctx, getDraftOrderByID, map[string]any{"id": gid}, &resp); err != nil {
		r.Logger.Error(err.Error(), zap.Any("gid", gid))
		return nil, err
	}

	if resp.DraftOrder == nil {
		r.Logger.Error("draft order not found", zap.String("id", gid))
		return nil, fmt.Errorf("draft order %s not found", gid)
	}

	return &resp, nil
}

// CreateOrder creates a order
func (r *repository) CreateOrder(
	ctx context.Context, input any,
) (*OrderCreateResponse, error) {
	var resp CreateOrderResponse
	if err := r.gql.Do(ctx, createOrder, input, &resp); err != nil {
		return nil, err
	}

	if resp.UserErrors != nil {
		r.Logger.Error("failed to create order", zap.Any("errors", resp.UserErrors))
		return nil, errors.New("failed to create order")
	}

	return &resp.OrderCreate.Order, nil
}

// CompleteDraftOrder completes a draft order and creates an order
func (r *repository) CompleteDraftOrder(ctx context.Context, draftOrderID string) error {
	if !strings.Contains(draftOrderID, DraftOrderKind) {
		draftOrderID = GID(DraftOrderKind, draftOrderID)
	}

	var resp CompleteDraftOrderResponse
	if err := r.gql.Do(ctx, draftOrderComplete, map[string]any{"id": draftOrderID}, &resp); err != nil {
		return err
	}

	if resp.UserErrors != nil {
		r.Logger.Error("failed to complete draft order", zap.Any("errors", resp.UserErrors))
		return errors.New("failed to complete draft order")
	}

	return nil
}

