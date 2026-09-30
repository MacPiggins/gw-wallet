package model

import "errors"

type Balance struct {
	Usd int64 `json:"USD"`
	Eur int64 `json:"EUR"`
	Rub int64 `json:"RUB"`
}

type Wallet struct {
	UserName string
	Balance  Balance
}

type WalletOps struct {
	Amount   int64  `json:"amount" binding:"required"`
	Currency string `json:"currency" binding:"required"`
}

var (
	ErrInvalidOperation = errors.New("invalid operation error")
)
