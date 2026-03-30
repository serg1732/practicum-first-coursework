package model

import "time"

const (
	ORDER_STATUS_REGISTERED = "REGISTERED"
	ORDER_STATUS_NEW        = "NEW"
	ORDER_STATUS_INVALID    = "INVALID"
	ORDER_STATUS_PROCESSING = "PROCESSING"
	ORDER_STATUS_PROCESSED  = "PROCESSED"
)

type Order struct {
	OrderId    string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *float64  `gorm:"accrual" json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type OrderDB struct {
	AccountId int64  `gorm:"account_id"`
	OrderId   string `gorm:"order_id"`
}

type OrderUpdate struct {
	Status string `json:"status"`
}

type BalanceDB struct {
	AccountId int64   `gorm:"account_id" json:"-"`
	Balance   float64 `gorm:"balance" json:"current"`
	Withdraw  float64 `gorm:"withdraw" json:"withdrawn"`
}

type Withdrawals struct {
	AccountId   int64     `gorm:"account_id" json:"-"`
	OrderId     string    `gorm:"order_id" json:"order"`
	Sum         float64   `gorm:"sum" json:"sum"`
	ProcessedAt time.Time `gorm:"processed_at" json:"processed_at"`
}
