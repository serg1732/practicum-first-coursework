package model

type BalanceResponse struct {
	Sum      float64 `json:"current"`
	Withdraw float64 `json:"withdrawn"`
}

type AccrualResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual"`
}
