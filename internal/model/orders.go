package model

import "time"

const (
	OrderStatusRegistered = "REGISTERED"
	OrderStatusNew        = "NEW"
	OrderStatusInvalid    = "INVALID"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusProcessed  = "PROCESSED"
)

type Order struct {
	OrderID    string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *float64  `gorm:"accrual" json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type OrderDB struct {
	AccountID int64  `gorm:"account_id"`
	OrderID   string `gorm:"order_id"`
}

type OrderUpdate struct {
	Status string `json:"status"`
}

type BalanceDB struct {
	AccountID int64   `gorm:"account_id" json:"-"`
	Balance   float64 `gorm:"balance" json:"current"`
	Withdraw  float64 `gorm:"withdraw" json:"withdrawn"`
}

type Withdrawals struct {
	AccountID   int64     `gorm:"account_id" json:"-"`
	OrderID     string    `gorm:"order_id" json:"order"`
	Sum         float64   `gorm:"sum" json:"sum"`
	ProcessedAt time.Time `gorm:"processed_at" json:"processed_at"`
}
