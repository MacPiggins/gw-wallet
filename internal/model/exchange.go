package model

// type Rates struct {
// 	UsdTo float32
// 	RUB
// }

type ExchangeRate struct {
	From string
	To   string
	Rate float32
}

type ExchangeOps struct {
	From   string `json:"from_currency" binding:"required"`
	To     string `json:"to_currency" binding:"required"`
	Amount int64  `json:"amount" binding:"required"`
}
