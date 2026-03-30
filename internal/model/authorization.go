package model

import "github.com/golang-jwt/jwt/v5"

type Accounts struct {
	ID       int64  `gorm:"id"`
	Login    string `gorm:"login" json:"login" binding:"required"`
	Password string `gorm:"password" json:"password" binding:"required"`
}

type Claims struct {
	AccountID int64 `json:"account_id"`
	jwt.RegisteredClaims
}
