package helpers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"bone_appetit_suscription/internal/models"
	dbModels "bone_appetit_suscription/pkg/db/models"
	"bone_appetit_suscription/pkg/shopify"
)

var manualPaymentGateway = []string{"Zelle", "Efectivo en la Entrega", "Pago Móvil", "Cashea"}

func CalculateNextDeliveryDate(currentDate time.Time, frequencyTag string) (time.Time, error) {
	weeks := 1
	if !strings.Contains(frequencyTag, "one_week") {
		frequency := strings.ReplaceAll(frequencyTag, "_weeks", "")
		if value, err := strconv.Atoi(frequency); err == nil {
			weeks = value
		}
	}

	nextDeliveryDate := currentDate.Add(time.Duration(weeks) * 7 * 24 * time.Hour)
	return nextDeliveryDate, nil
}

func GetLineItemsForEmail(lineItems []shopify.LineItemsNode) []string {
	var items []string
	for _, lineItem := range lineItems {
		if lineItem.Node.Quantity == 0 {
			continue
		}
		items = append(items, fmt.Sprintf("%dx %s\n\n", lineItem.Node.Quantity, lineItem.Node.Name))
	}
	return items
}

func GetVarsForConfirmationOrderEmail(
	varsForEmail models.ConfirmationOrderEmailVars,
) map[string]any {
	vars := make(map[string]any)
	if varsForEmail.CustomerName != "" {
		vars["display_name"] = varsForEmail.CustomerName
	}
	if varsForEmail.OrderNo != "" {
		vars["order_no"] = varsForEmail.OrderNo
	}
	if varsForEmail.NextDeliveryDate != "" {
		vars["date"] = varsForEmail.NextDeliveryDate
	}
	if varsForEmail.Method != "" {
		vars["method"] = varsForEmail.Method
	}
	if varsForEmail.Frequency != "" {
		vars["frequency"] = varsForEmail.Frequency
	}
	if len(varsForEmail.Items) > 0 {
		vars["items"] = strings.Join(varsForEmail.Items, "\n")
	}
	if varsForEmail.CustomerAddress != nil {
		vars["address"] = fmt.Sprintf(
			"%s\n%s\n%s\n%s\n%s\n%s",
			varsForEmail.CustomerAddress.Address1,
			varsForEmail.CustomerAddress.Address2,
			varsForEmail.CustomerAddress.City,
			varsForEmail.CustomerAddress.Province,
			varsForEmail.CustomerAddress.Country,
			varsForEmail.CustomerAddress.Zip,
		)
	}
	if varsForEmail.IsDraft {
		vars["link"] = varsForEmail.OrderURL
	} else {
		vars["createOrder"] = varsForEmail.OrderURL
	}

	return vars
}

// IsManualPaymentGateway checks if the payment gateway is manual
func IsManualPaymentGateway(paymentGateway []string) bool {
	for _, pg := range paymentGateway {
		if slices.Contains(manualPaymentGateway, strings.TrimSpace(pg)) {
			return true
		}
	}

	return false
}

// AddLog adds a log entry for a subscription order
func AddLog(ctx context.Context, tx *gorm.DB, log models.Log, dbOrderID string) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}

	err := tx.WithContext(ctx).
		Create(&dbModels.SuscriptionLog{
			Type:         log.Type,
			Message:      log.Message,
			UpdatedField: log.UpdatedField,
			NewValue:     fmt.Sprintf("%v", log.NewValue),
			OldValue:     fmt.Sprintf("%v", log.PreviousValue),
			OrderID:      dbOrderID,
		}).Error
	if err != nil {
		fmt.Printf("error adding log: %v\n", err)
		return err
	}

	return nil
}

// IsAutomaticSubscription checks if the order is an automatic subscription
func IsAutomaticSubscription(tags []string) bool {
	if slices.Contains(tags, "appstle_subscription_recurring_order") || slices.Contains(tags, "appstle_subscription_first_order") {
		return true
	}

	return false
}

// IsShopifyPaymentGateway checks if the payment gateway is Shopify Payments
func IsShopifyPaymentGateway(paymentGateway []string) bool {
	if slices.Contains(paymentGateway, "shopify_payments") {
		return true
	}

	return false
}

func GenerateAuthToken(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// SameDay checks if two timestamps are on the same day.
func SameDay(t1, t2 time.Time) bool {
	if t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day() {
		return true
	}
	return false
}
