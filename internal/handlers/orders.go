package handlers

import (
	"bone_appetit_suscription/internal/domains"
	"bone_appetit_suscription/internal/models"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	service domains.OrderService
}

func NewOrderHandler(service domains.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

// ScheduleNextOrder mark order as paid and create the next order
func (h *OrderHandler) ScheduleNextOrder(c *gin.Context) {
	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.IsPaid = false
	req.OrderID = fmt.Sprintf("#%s", req.OrderID)
	if err := h.service.ScheduleNextOrder(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order created successfully"})
}

// CreateOnDemandOrderHandler handles the creation of on-demand orders
func (h *OrderHandler) CreateOnDemandOrderHandler(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		fmt.Println("order_no is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_no is required"})
		return
	}

	invoiceURL, err := h.service.CreateOnDemandOrderProcess(context.Background(), orderNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if invoiceURL == "" {
		fmt.Printf("Failed to create on-demand order %s\n", orderNo)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Read the .confirm.html file
	htmlBytes, err := os.ReadFile("confirm.html")
	if err != nil {
		fmt.Printf("Error reading .confirm.html: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	htmlContent := string(htmlBytes)
	// Replace {{checkout}} with invoiceURL
	htmlContent = strings.ReplaceAll(htmlContent, "{{checkout}}", invoiceURL)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}
