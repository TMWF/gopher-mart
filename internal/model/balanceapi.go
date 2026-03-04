package model

import "time"

type GetBalanceResponseModel struct {
	Current   float64 `json:"current"`
	WithDrawn float64 `json:"withdrawn"`
}

type WithDrawBalanceRequestModel struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type GetUserWithdrawalsResponseModel struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
