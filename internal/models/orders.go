package models

import "time"

type CreateOrderRequest struct {
	OrderID string `json:"orderId" binding:"required"`
	Receipt string `json:"receipt"`
	Method  string `json:"method"`
	IsPaid  bool   `json:"isPaid"`
}

type ConfirmationOrderEmailVars struct {
	IsDraft          bool             `json:"is_draft"`
	OrderURL         string           `json:"order_url"`
	Items            []string         `json:"items"`
	OrderNo          string           `json:"order_no"`
	NextDeliveryDate string           `json:"next_delivery_date"`
	Method           string           `json:"method"`
	Frequency        string           `json:"frequency"`
	CustomerAddress  *CustomerAddress `json:"customer_address"`
	CustomerName     string           `json:"customer_name"`
}

type CustomerAddress struct {
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	Province string `json:"province"`
	Country  string `json:"country"`
	Zip      string `json:"zip"`
}

type Log struct {
	Type          string    `json:"type"`
	Message       string    `json:"message"`
	NewValue      any       `json:"newValue"`
	UpdatedAt     time.Time `json:"updatedAt"`
	UpdatedField  string    `json:"updatedField"`
	PreviousValue any       `json:"previousValue"`
}

