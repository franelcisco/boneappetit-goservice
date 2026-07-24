package models

import "time"

type SuscriptionLog struct {
	ID           int64     `json:"id" gorm:"column:id"`
	Type         string    `json:"type" gorm:"column:type"`
	Message      string    `json:"message" gorm:"column:message"`
	NewValue     string    `json:"newValue" gorm:"column:new_value"`
	OldValue     string    `json:"oldValue" gorm:"column:old_value"`
	UpdatedField string    `json:"updatedField" gorm:"column:updated_field"`
	CreatedAt    time.Time `json:"createdAt" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	OrderID      string    `json:"orderId" gorm:"column:order_id"`
}

func (SuscriptionLog) TableName() string {
	return "suscription_logs"
}
